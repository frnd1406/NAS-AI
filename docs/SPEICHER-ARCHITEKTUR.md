# NAS.AI – Speicher-Architektur & Flexibilitäts-Design

> **Status:** Entwurf v0.6 · **Datum:** 2026-08-14 · (v0.4: §12 App-Patching · v0.5: SSH+FIDO2-Trigger, Updater komplett & E2E-getestet · v0.6: verifizierter Git-Stand + Tagging-Plan — Tags gebündelt später, §11)
> **Zweck:** Referenz-Dokument, bevor Code geändert wird. Legt fest, wie das Backend
> Laufwerke erkennt, den Tresor verwaltet und RAID-Spiegelung anbietet — so flexibel,
> dass neue Hardware (eigene Waveshare-artige Boards, mehrere SATA/SSD/NVMe) ohne
> Chaos und ohne Backend-Umbau angebunden werden kann.
> **Ablageort später:** `NasServer/docs/SPEICHER-ARCHITEKTUR.md`

---

## 0. Grundprinzip

**Das Backend darf nie ein bestimmtes Gerät „kennen".** Es arbeitet ausschließlich mit
**logischen Rollen** (heiß / groß / Tresor / Transfer). Welches physische Laufwerk hinter
einer Rolle steckt, entscheidet die Konfiguration — nicht der Code. Die Hardware ist
austauschbar; die Rolle bleibt.

Merksatz: **Rollen im Code, Geräte in der Konfig.**

---

## 1. Ist-Zustand (Stand 2026-08-14)

### 1.1 Was heute schon flexibel ist
Die Flexibilität sitzt aktuell in der **Docker-Volume-Schicht**, nicht im Go-Code:
Der Container hat feste interne Pfade, die compose auf Host-Geräte mappt.

```
Container-Pfad          ← compose (.env.pi)         → physisches Gerät
/var/lib/postgresql/data  ${SSD_PATH}/postgres        NVMe / SSD
/mnt/data                 ${RAID_PATH}/data           (heute NVMe, später HDD-RAID)
/mnt/backups              ${RAID_PATH}/backups        …
/media/user (Tresor)    ${RAID_PATH}/encrypted      …
/app/tls (Cert)           ${SSD_PATH}/tls:ro          …
```

**Deshalb lief die NVMe-Migration ohne eine Code-Zeile** — dem Backend war egal, ob
`/mnt/data` auf SD oder NVMe liegt. Für „Gerät hinter einem Mount tauschen" ist das gut.

### 1.2 Grenzen & Baustellen
1. **Nur 2 Tiers** (`SSD_PATH` schnell, `RAID_PATH` groß). Mehr Abstufungen (heiß/groß/
   Transfer/Tresor getrennt) lassen sich heute nicht ausdrücken.
2. **`/mnt/data` ist ~15× hardcodiert** — u. a. in SQL (`search.go:100` `LIKE '/mnt/data/%'`)
   und im nginx-X-Accel-Rewrite (`content_delivery_service.go:116`). Koppelt DB-Inhalt &
   Code an einen Magic-String.
3. **Vault-Pfad-Bug (wichtig, sicherheitsrelevant):**
   - `secure_ai_feeder.go:170` liest `ENCRYPTED_STORAGE_PATH` ✔
   - `server.go:253` **ignoriert** die Variable und nagelt `/media/user/DEMO` fest ✗
   - → zwei Komponenten, zwei Verhalten für denselben Tresor. Muss **eine** Quelle werden.
4. **Keine Laufwerks-Erkennung** — das System weiß nicht, ob ein Mount HDD, SSD oder NVMe ist.
5. **Kein echtes RAID** — `RAID_PATH` ist bislang ein Ordner, kein mdadm-Array.

---

## 2. Ziel-Architektur: logische Tiers

| Rolle | Typ-Empfehlung | Inhalt | Env (Vorschlag) |
|---|---|---|---|
| **HOT** | SSD / NVMe | DB, Redis, Chroma-Index, TLS, aktive Metadaten | `STORAGE_HOT_PATH` |
| **BULK** | HDD-RAID | große Dateien, Medien, Backups, Archiv | `STORAGE_BULK_PATH` |
| **VAULT** | isoliert (s. §4) | verschlüsselter Tresor | `VAULT_PATH` |
| **TRANSFER** | SSD | schneller Zwischenspeicher / Datenübertragung (künftige 2×2TB-SSD) | `STORAGE_TRANSFER_PATH` |

Der Container behält stabile interne Mounts (`/mnt/data`, `/mnt/backups`, `/media/vault`,
`/mnt/transfer`); im Go-Code ersetzt eine zentrale `StorageConfig` (einmal aus Env gelesen)
alle Hardcodes. Neue Tiers = neue Env + neuer Mount, **kein Code-Umbau**.

---

## 3. Anforderung 1 — Laufwerks-Erkennung (HDD/SSD) → Maßnahmen

**Ziel:** Das System erkennt selbstständig, ob ein Laufwerk HDD oder SSD/NVMe ist, und
schlägt daraufhin passende Maßnahmen/Einstellungen vor.

### 3.1 Wie erkannt wird (Linux, keine Rate-Werte)
- **Typ (dreht sich?):** `/sys/block/<dev>/queue/rotational` → `1` = HDD, `0` = SSD/NVMe
- **Anschluss:** `lsblk -dno NAME,ROTA,TRAN,MODEL,SIZE,SERIAL` → `TRAN` = `sata|nvme|usb`
- **Gesundheit:** `smartctl -H -A` (SMART) — bei HDD mechanische Werte, bei SSD Verschleiß/TBW
- **RAID-Mitgliedschaft:** `/proc/mdstat`, `mdadm --detail`
- **Belegung:** aktueller Mountpoint/FS via `lsblk -f`

### 3.2 Klassifizierung
Ergebnis pro Gerät: `{ name, typ: HDD|SSD|NVMe|USB, größe, modell, serial, smart_status,
rolle_vorschlag, gemountet_als }`.

### 3.3 Typ → Maßnahmen (was das System anbieten soll)

| Aspekt | SSD / NVMe | HDD |
|---|---|---|
| Empfohlene Rolle | HOT / TRANSFER | BULK / Archiv |
| TRIM / `fstrim.timer` | **aktivieren** | entfällt |
| Spindown / APM (`hdparm -S`) | entfällt | konfigurierbar (Idle-Standby) |
| SMART-Überwachung | Verschleiß-%, TBW, Reserve-Blöcke | Reallocated Sectors, Spin-Retry, Temp |
| Defrag | **nie** | entfällt (ext4 unkritisch) |
| Staggered Spin-Up | entfällt | **empfohlen** (mehrere HDDs) |
| Power-Loss-Risiko | hoch ohne PLP → USV | mittel → USV |
| Power-Management | ASPM/APST bewusst setzen¹ | Link-PM bewusst setzen¹ |

> ¹ **Lektion NVMe-Ausfall:** Aggressives Power-Saving warf die NVMe aus dem System. Für
> Platten im RAID gilt dasselbe — ein zu sparsamer Controller kann eine Platte aus dem
> Array kippen. Power-Management pro Laufwerk bewusst konfigurieren, nicht Default lassen.

### 3.4 Ablauf (Settings-Flow)
1. System scannt Laufwerke → zeigt erkannte Geräte + Typ + SMART.
2. System **schlägt** Rolle + typgerechte Maßnahmen **vor**.
3. Nutzer bestätigt oder überschreibt → Einstellung wird persistiert.
4. Maßnahmen werden angewandt (TRIM-Timer, Spindown, SMART-Monitoring, RAID s. §5).

---

## 4. Anforderung 2 — Tresor (Vault) konfigurierbar

**Ziel:** Der Tresor ist frei einstellbar — Ort, Granularität und ob er gesichert wird.

### 4.1 Platzierungs-Modi (wählbar)
| Modus | Bedeutung | Vorteil | Hinweis |
|---|---|---|---|
| **Ganze Platte** | dediziertes Laufwerk nur für Tresor | stärkste physische Isolation | ein Laufwerk „geopfert" |
| **Eigene Partition** | separate Partition auf einer Platte | isoliert ohne Extra-Laufwerk | Partitionierung nötig |
| **Nur File** | verschlüsselter Container als Datei im FS | am flexibelsten, einfach umziehbar | teilt Medium mit anderen Daten |

### 4.2 Backup des Tresors
- **Umschaltbar:** Tresor-Backup **an / aus** (bewusste Entscheidung des Nutzers).
- **Wenn an:** eigenes **Backup-Ziel** wählbar (anderes Medium/Tier, idealerweise off-box).
- **Backup-Form:** verschlüsselter Blob/Datei — Tresor bleibt **auch im Backup verschlüsselt**
  (nie Klartext ablegen).

### 4.3 Sicherheits-Hard-Rules (nicht überschreibbar)
- 🔒 **Der Tresor ist für MCP/Agent IMMER tabu** — egal welche Rechte eingestellt sind.
- 🔒 MCP/KI darf **nur `public`-Daten** lesen/übertragen, **nie** Vault-Daten.
- 🔒 Datenklasse `public` vs. `vault` wird **im Service-Layer strukturell erzwungen**
  (nicht „per Einstellung aus", sondern der Pfad kann Vault-Daten gar nicht erreichen).
- 🔒 DEK bleibt nur im RAM; nach Neustart Vault erneut entsperren.

### 4.4 Konfig
Einheitliche, env-getriebene Quelle (behebt §1.2 Punkt 3):
`VAULT_PATH`, `VAULT_MODE = disk|partition|file`, `VAULT_BACKUP_ENABLED = true|false`,
`VAULT_BACKUP_PATH`. **`server.go:253` und `secure_ai_feeder.go` müssen dieselbe Quelle nutzen.**

---

## 5. Anforderung 3 — RAID / Spiegelung: Level-Wahl

**Ziel:** Die Spiegelungs-Stufe ist wählbar. Umsetzung via **mdadm** (Software-RAID, läuft
neben dem Docker-Stack, unabhängig vom Controller).

### 5.1 Realistische Level (ehrliche Einordnung)
| Level | Min. Platten | Übersteht Ausfall | Nutzkapazität | Einsatz |
|---|---|---|---|---|
| RAID 0 | 2 | **keine** | 100 % | nur Tempo, 1 Platte tot = alles weg — **nicht für NAS-Daten** |
| **RAID 1** | 2 | 1 Platte | 50 % | Spiegel — **Test-Phase (2 Platten)**, einfach & sicher |
| RAID 5 | 3 | 1 Platte | (n−1)/n | gute Kapazität; „Write-Hole" ohne USV (du hast eine) |
| **RAID 6** | 4 | **2 Platten** | (n−2)/n | **Zukunft 4×5TB** — bei großen HDDs Pflicht (Rebuild-Stress) |
| RAID 10 | 4 | 1 je Spiegel | 50 % | Tempo + Redundanz, schnellster Rebuild |

> **Hinweis zu „Stufe 1, 2, 3":** RAID 2 und RAID 3 sind historisch und praktisch
> ausgestorben (mdadm kann RAID 2 nicht einmal). Die sinnvolle Leiter ist **0 / 1 / 5 / 6 / 10**.
> Ich empfehle, im UI genau diese anzubieten und je nach Plattenzahl zu filtern.

### 5.2 Empfehlung je Ausbaustufe
- **Jetzt (2 HDD, Test):** RAID 1
- **Zukunft (4×5TB, 20 TB):** RAID 6 — überlebt 2 gleichzeitige Ausfälle; wichtig, weil beim
  Rebuild großer HDDs die Belastung gern eine zweite Platte killt.
- **Transfer-SSDs (2×2TB):** je nach Zweck — reiner Scratch → RAID 0 (Tempo); sollen sie sicher
  sein → RAID 1.

### 5.3 mdadm-Details (Pflicht)
- **Write-Intent-Bitmap** an → nach unsauberem Abschalten schneller Re-Sync statt Full-Rebuild.
- **Monitoring:** `mdadm --monitor` + SMART; Alarm bei `degraded`/`faulty`.
- **Power-Management** der Array-Platten bewusst setzen (§3.3 ¹).

### 5.4 RAID ≠ Backup (bleibt gültig)
RAID schützt gegen **Platten-Ausfall**, nicht gegen Löschen, FS-Korruption, Ransomware,
Blitz/Feuer/Diebstahl. Deshalb **zusätzlich** eine getrennte Off-Box-Kopie (4B-Link /
externes Medium). RAID = Verfügbarkeit, Backup = getrennte Kopie.

---

## 6. Konfig-Schema (Zusammenfassung)

**Env (Infrastruktur, pro Deployment):**
```
STORAGE_HOT_PATH=/mnt/hot
STORAGE_BULK_PATH=/mnt/bulk
STORAGE_TRANSFER_PATH=/mnt/transfer
VAULT_PATH=/media/vault
VAULT_MODE=file|partition|disk
VAULT_BACKUP_ENABLED=true|false
VAULT_BACKUP_PATH=/mnt/bulk/vault-backup
ALLOWED_STORAGE_ROOTS=/mnt/hot,/mnt/bulk,/mnt/transfer   # Sicherheits-Zaun
```
**Settings-DB (nutzer-einstellbar, zur Laufzeit):**
erkannte Laufwerke + Rolle, Maßnahmen (TRIM/Spindown/SMART), RAID-Level je Array,
Tresor-Modus & -Backup-Toggle.

**Code-Änderung (Kern):** zentrale `StorageConfig{HotRoot, BulkRoot, VaultRoot, TransferRoot,
BackupRoot, AllowedRoots}` — ersetzt die ~15 `/mnt/data`-Hardcodes und den Vault-Hardcode.
Fernziel: **relative Pfade in der DB** statt absoluter → komplett mount-unabhängig.

---

## 7. Sicherheits-Grundregeln (aus Nutzer-Vorgaben, verbindlich)
1. MCP/Agent-Rechte pro Aktion einstellbar — **Tresor immer tabu** (Hard-Rule).
2. Firewall-Steuerung nur mit **Passwort + YubiKey**, **nie** über MCP/Agent.
3. USV = automatisches Backup während des Ladens (später, eigene Platine).
4. MCP-KI nur **public** lesen/übertragen, nie Vault; Datenklasse im Service-Layer erzwungen.

---

## 8. Umsetzungs-Phasen (Vorschlag, später)
- **P0 – Fix + Fundament:** Vault-Pfad-Bug beheben (eine Env-Quelle); `StorageConfig` einziehen,
  `/mnt/data`-Hardcodes darüber leiten. (kleiner, hoher Wert)
- **P1 – Erkennung:** Laufwerks-Inventar-Service (Typ/SMART/Transport) + Settings-Anzeige.
- **P2 – RAID:** mdadm-Verwaltung (Level-Wahl, Bitmap, Monitoring) + `raid/`-Daten migrieren.
- **P3 – Tresor-Optionen:** Modus (disk/partition/file) + Backup-Toggle + -Ziel.
- **P4 – Datenklasse:** public/vault strukturell im Service-Layer, MCP-Isolation verifizieren.
- **Fernziel:** relative DB-Pfade; Tiers HOT/BULK/TRANSFER produktiv.

> Umsetzungs-Hinweis: Repo liegt unter `Dokumente` → CFA blockt Git-Writes. Vor Code-Phase
> Repo rausziehen **oder** `git.exe`/Build-Tools erlauben.

---

## 9. Offene Fragen / Entscheidungen
- [ ] Sollen Tiers **fest** (HOT/BULK/TRANSFER) oder **frei benennbar** sein?
- [ ] Tresor-Default-Modus? (Vorschlag: `file` — am flexibelsten, leicht umziehbar)
- [ ] RAID-Verwaltung im **WebUI** oder nur per CLI/Script (Sicherheit vs. Komfort)?
- [ ] Off-Box-Ziel final: 4B-Link vs. externes rotiertes Medium vs. beides?
- [ ] Migration auf relative DB-Pfade: jetzt mitdenken oder als späteres Extra?

---

## 10. Wettbewerb & Differenzierung (Recherche 2026-08-14)

Vergleich mit Synology, QNAP, Western Digital, Netgear.

### 10.1 Was wir übernehmen sollten (fehlt uns noch)
| Feature | Warum | Vorbild |
|---|---|---|
| **Btrfs/ZFS-Snapshots (read-only/immutable)** | stärkster Ransomware-Schutz — Rollback, den Angreifer nicht verschlüsseln können | Synology Btrfs, QNAP QuTS/ZFS |
| **Immutable/Snapshot-Backup** (WORM) | getrennte, gesperrte Kopie ergänzt RAID | Hyper Backup |
| **Daten-Integrität / Checksums + Scrub** | Bit-Rot-Schutz für 20-TB-HDD-Zukunft | ZFS/Btrfs-Scrub |
| **SMART-/Health-Dashboard** | deckt Anforderung §3 (Laufwerks-Erkennung) ab | alle |
| **On-device Foto-/Objekt-KI** | unser KI-Kern — aber mit Vault-Tabu | Synology Photos, QNAP QuMagie |
| **Kamera-Hub on-device (ohne Fremd-Cloud)** | Nutzer hat Kamera; späteres Modul | Synology CC400W |

### 10.2 Was wir per Design schon besser lösen (Vorsprung halten)
- **Kein Internet-Exposure** → der #1-Angriffsvektor der anderen (DeadBolt-Ransomware bei QNAP, unauth. RCE **CVE-2025-30247** bei WD) fällt bei uns weg: LAN-only, ufw-Whitelist, kein Port-Forward.
- **Echtes Zero-Knowledge** (AES-256-GCM, Argon2id, DEK nur im RAM) → die Lücke, die Synology (eCryptFS, Keys nicht widerrufbar) und QNAP (Auto-Mount speichert Key) *nicht* schließen.
- **Kein Vendor-Cloud / kein Remote-Kill** → uns kann niemand fern-wipen (WD My-Book-Live-Massen-Wipe 2021) oder abschalten (Netgear ReadyCLOUD).
- **Kein Hardware-Lock-in** → eigene Hardware, jede Platte, eigene Root-CA (vgl. Synology Drive-Lock-in-Debakel 2025).

### 10.3 Warnungen (Kehrseite unseres Ansatzes)
- **Patching liegt bei UNS.** Netgear-Lektion: Abandonment = Risiko. `unattended-upgrades` läuft — diszipliniert halten.
- **Zertifikatsprüfung gezielt gegentesten** — Synology hatte MITM durch fehlerhafte Cert-Validierung (**CVE-2026-40539**); unsere Root-CA/WebPKI-Umstellung genau daraufhin prüfen.

---

## 11. Backlog — später erledigen (Gesamtstand 2026-08-14)

> Priorität: 🔴 hoch · 🟡 mittel · 🟢 später. Quelle in Klammern.

### ⭐ Patching — HÖCHSTE Priorität (Nutzer-Vorgabe 2026-08-14)
Grund: selbst gehostet → ungepatchtes System = größtes Risiko (Netgear-Abandonment-Lektion §10.3).
- [x] **NAS (.81) voll gepatcht** — 0 offen; Docker auf 29.7.2 (Stack ok); unattended-upgrades war schon aktiv
- [x] **Firewall (.82) voll gepatcht** — 76 Updates (13 Security) eingespielt; **Kernel 6.18.34→6.18.39** (Reboot erledigt); AdGuard ok
- [x] **Firewall: unattended-upgrades nachinstalliert + aktiviert** — hatte VORHER GAR KEIN Auto-Patching (die eigentliche Lücke!)
- [ ] 🔴 **App-gesteuertes Patching bauen** (Desktop-App als Steuerstand) — ersetzt jeden Script-/Cron-Notbehelf. Volle Spezifikation → §12.
- [ ] 🔴 **Off-Box-DB-Backup ist wieder MANUELL** (`C:\Users\user\nas-backups\pull-nas-db.sh` von Hand) — die Automatik wurde entfernt (s. u.); künftig ins App-Feature integrieren.
- [x] **Docker-Images patchen** — `docker`-Aktion im `nas-updater` (postgres+redis pull+recreate) deployt + getestet 2026-08-14
- [ ] 🟢 **rpt-Kernel/Firmware:** werden NICHT von unattended-upgrades erfasst → App muss sie separat erkennen (Kernel-Base-Vergleich) + Reboot anbieten

> **Notbehelf gebaut & wieder entfernt (2026-08-14):** Pi-Cron-Sensor (`patch-check.sh`) + Windows-Task (Ballon-Meldung + Off-Box-Pull) waren fertig und getestet, auf Nutzer-Wunsch aber komplett zurückgebaut zugunsten der App-Lösung (§12). `unattended-upgrades` bleibt aktiv. Der Windows-Task hatte einen `ssh.exe`-Hänger im Scheduler-Kontext — **Lektion für später: bei ssh aus nicht-interaktivem Kontext immer `-n` (stdin von /dev/null)**, sonst blockiert es sporadisch.

### Speicher / dieses Design
- [ ] 🔴 **Vault-Pfad-Bug:** `server.go:253` muss `ENCRYPTED_STORAGE_PATH` nutzen (eine Quelle wie AI-Feeder) (§1.2, §4.4)
- [ ] 🟡 **`StorageConfig` einziehen**, ~15× `/mnt/data`-Hardcodes darüber leiten (§2, §6)
- [ ] 🟡 **Laufwerks-Erkennung** (HDD/SSD/NVMe, SMART, Transport) + Settings-Flow (§3)
- [ ] 🟡 **RAID via mdadm:** Level-Wahl, Write-Intent-Bitmap, Monitoring; jetzt RAID1, Zukunft RAID6 (§5)
- [ ] 🟡 **Tresor-Optionen:** Modus disk/partition/file + Backup-Toggle + -Ziel (§4)
- [ ] 🟢 **Datenklasse public/vault** strukturell im Service-Layer erzwingen, MCP-Isolation verifizieren (§4.3)
- [ ] 🟢 **Relative DB-Pfade** statt absoluter (mount-unabhängig) (§6)
- [ ] 🟢 **Offene Design-Fragen** aus §9 mit Nutzer klären

### Aus Wettbewerbs-Recherche (neu)
- [ ] 🔴 **Btrfs/ZFS-Snapshots (immutable)** als Ransomware-Schutz — fehlt uns wirklich (§10.1)
- [ ] 🟡 **Immutable Off-Box-Backup** (WORM/gesperrt) (§10.1)
- [ ] 🟡 **SMART-Health-Dashboard** (deckt sich mit Laufwerks-Erkennung) (§10.1)
- [ ] 🟢 **Scrub/Checksums** für 20-TB-Zukunft; **Foto-KI** on-device; **Kamera-Hub** (§10.1)
- [ ] 🟡 **Cert-Validierung gegen MITM gegentesten** (CVE-2026-40539-Stil) (§10.3)

### Backup / Betrieb (dringlich)
- [ ] 🔴 **Off-Box-DB-Backup automatisieren:** `pull-nas-db.sh` in Windows-Task-Scheduler (db-backups liegen wieder mit der DB auf demselben Medium!)
- [ ] 🟢 **USV/NUT** einspielen + Stecker-Zieh-Test — sobald USV da (Runbook `~/nut-prep/`)
- [ ] 🟢 **SATA-HAT/RAID-Migration** — sobald Multi-PCIe-Board + Platten da (RAID1 + `raid/`-Daten migrieren + Off-Box-Link zum 4B)

### Repo / Release / Tagging (Stand 2026-08-14 verifiziert)
**Entscheidung: Tags gebündelt setzen, wenn der Unterbau sauber ist — NICHT einzeln jetzt.**
- [x] NasServer-Security-Backports **sind bereits committet** (branch `fix/dev-dockerfile-go125-crlf`): TLS-Prod `3ff4e31`, Host-Mount-Entfernung `db7f523`, BIND_ADDR `8589bd5`, Backup-Script `1d4c375`, Tresor-Volume `a1da18d`.
- [x] nas-desktop CA-Validierung **ist committet** (`9e1788c`) — aber **UNGETAGGT** (liegt nach letztem Tag `v0.1.0-beta.2.2.1`).
- [ ] 🟡 **NasServer: 7 unerwartete Löschungen** (`.agents/…`, `.codex/…`) im Working-Tree klären (nicht aus dieser Session) — vor jedem Commit.
- [ ] 🟡 **fix-Branch `fix/dev-dockerfile-go125-crlf` → main** mergen (NasServer, Tag-Schema `v2.x`).
- [ ] 🟡 **Updater-Scripts + dieses Dokument nach NasServer backporten** (`infrastructure/` + `docs/`) — dann versioniert.
- [ ] 🟡 **nas-desktop:** nächsten **kleinen** Tag setzen (CA-Feature) + Release-Build + Signatur (YubiKey). Nummer offen (z. B. `v0.1.0-beta.2.2.2`).
- [ ] 🟡 **Root-CA in Windows-Trust** importieren (`certutil -addstore -user Root`) für Browser/WebUI.
- [ ] 🟢 **CFA:** `NasServer` aus `Dokumente` rausziehen oder `git.exe` erlauben — **Voraussetzung** für alle NasServer-Commits/Tags oben.
- Hinweis: Signieren nur im Nutzer-Terminal (YubiKey-Touch) bzw. via windows-mcp `git_signed_commit`.

### Funktion / Ausbau
- [ ] 🟡 **End-to-End-App-Test:** Dashboard, Upload/Download, Admin-Bereich
- [ ] 🟢 **webui + chromadb** dazu; **Phase-2-AI-Refactor** (Ollama raus, ChromaDB)
- [ ] 🟢 **Prod-Härtung:** /host-Mount verkleinern

---

## 12. Feature-Spec: App-gesteuertes Patching (Entscheidung 2026-08-14)

**Ziel/UX (Nutzer-Vorgabe):** Die Desktop-App ist der Steuerstand. Sie zeigt ein Banner
„Update verfügbar", der Nutzer klickt → **beide Pis** werden aktualisiert; ist ein Reboot
nötig, rebootet der Pi und die **App wartet mit Spinner**, bis er wieder oben ist, dann „fertig".

### 12.1 Sicherheitsmodell (verbindlich, REVIDIERT 2026-08-14)
**Kein neuer Netzwerk-Daemon** (ein lauschender Root-Dienst wäre selbst Angriffsfläche — genau der
Vektor, der QNAP/WD zerlegt hat). Stattdessen **Trigger über den bereits gehärteten SSH-Kanal**
(key-only, ufw nur PC):
- **`nas-updater.sh` (Host-Script, root) pro Pi** — kann NUR `status` · `upgrade` · `reboot`. Kein Vault-/Datenzugriff.
- **Forced-Command-Dispatcher** (`nas-updater-dispatch`): ein dedizierter SSH-Key darf ausschließlich
  diese drei Aktionen auslösen, **keine Shell** (`restrict` + `command=`).
- **Gate = dedizierter FIDO2-YubiKey-SSH-Key** (`ed25519-sk`, resident, **verify-required = PIN**, Touch-Zwang)
  → das ist „YubiKey + Passwort (PIN)" pro Aktion, **ohne einen einzigen neuen Port**.
- **Container-API bleibt unprivilegiert** (`nobody`, cap_drop ALL) — reicht höchstens Status read-only durch, führt NIE Host-Kommandos aus.
- **Nur auf ausdrückliche Nutzer-Aktion** (nie autonom); **nie** für die MCP-/KI-Schicht erreichbar (Hard-Rule §7.1/§7.4).
- **Beide Pis gleich**, jede Aktion mit YubiKey-PIN+Touch (Nutzer-Entscheidung — erfüllt zugleich Firewall-Hard-Rule §7.2).

### 12.2 Komponenten & Umsetzungsstand
1. ✅ **`nas-updater.sh`** (`/usr/local/sbin/`, beide Pis) — deployt + getestet (status/upgrade No-Op ok).
2. ✅ **`nas-updater-dispatch`** (`/usr/local/sbin/`, beide Pis) — deployt; Denied-Test (Fremdkommando) greift.
3. ✅ **Dedizierter FIDO2-Key** (`id_ed25519_sk_updater`, resident, verify-required, +Datei-Passphrase) —
   generiert + Pubkey mit `restrict,verify-required,command="…dispatch"` in `authorized_keys` beider Pis.
   **End-to-End verifiziert 2026-08-14:** `status` NAS+FW liefert Report (PIN+Touch), Fremdbefehl wird abgelehnt.
   → **Sichere Trigger-Kette komplett & getestet.** Nutzbar schon per CLI:
   `ssh -i ~/.ssh/id_ed25519_sk_updater <pi> {status|upgrade|docker|reboot}`.
   ✅ **`docker`-Aktion** (2026-08-14): zieht+erneuert nur externe Images (postgres+redis), lässt die lokal
   gebaute `api` in Ruhe, Firewall skippt sauber. Getestet — pgvector:pg16 aktualisiert, Stack danach healthy.
   (Rebuild der lokalen `api` = App-Deploy, gehört NICHT in den Patcher.)
4. ⬜ **API-Statusdurchreiche (NAS)** — zeigt Updater-`status` read-only; löst nichts aus.
5. ⬜ **Desktop-App (Tauri/Rust)** — Banner, „Jetzt aktualisieren"-Button (löst SSH-Trigger → YubiKey blinkt/PIN),
   Fortschritt, **Spinner mit Health-Polling** nach Reboot. (CFA/Build, siehe §12.5)

### 12.3 Ablauf
1. Status (vom `nas-updater`) → Banner „Update verfügbar (X Security, Kernel Y)".
2. Klick → **YubiKey + Passwort** → App ruft `nas-updater upgrade` auf beiden Pis.
3. Reboot nötig? Updater rebootet → App **pollt Health** bis erreichbar → Spinner stoppt → „fertig, Kernel Z".
   - **NAS:** App verliert dabei die eigene Backend-Verbindung (API läuft auf dem NAS) → Spinner wartet auf deren Rückkehr.
   - **Firewall:** separater Erreichbarkeits-Check (App ist mit ihr normal nicht verbunden).

### 12.4 Offene Detailfragen
- [ ] Konkreter Auth-Mechanismus Updater ↔ App (mTLS mit unserer Root-CA? Token? + wie genau bindet die App den YubiKey ein).
- [ ] Firewall-Erreichbarkeit aus der App (direkter Health-Port des `nas-updater` auf .82, ufw-Freigabe nur für PC).
- [ ] Docker-Image-Updates (`compose pull`+rebuild) mit ins Feature? (empfohlen: ja, als eigener Schritt).
- [ ] Staggered/Reihenfolge: erst NAS, dann Firewall? (Firewall zuletzt, damit die App-Verbindung so lang wie möglich steht.)

### 12.5 Voraussetzung
Code-Änderungen (nas-desktop + NasServer) — beide Repos liegen unter `Dokumente` → CFA blockt Git-Writes;
vorher rausziehen/erlauben. Commits signiert (YubiKey). Siehe [[nas-desktop-build-env]].

---

## Anhang — relevante Code-Stellen (Ist)
| Datei:Zeile | Inhalt |
|---|---|
| `server.go:246` | `NewLocalStore("/mnt/data")` — hardcodiert |
| `server.go:253` | `encryptedStoragePath := "/media/user/DEMO"` — **ignoriert Env (Bug)** |
| `secure_ai_feeder.go:158/170` | Default `/media/user/DEMO`, liest aber `ENCRYPTED_STORAGE_PATH` ✔ |
| `search.go:100/113` | SQL `LIKE '/mnt/data/%'` — Magic-String in Query |
| `content_delivery_service.go:116` | `/mnt/data` → `/protected-files` (X-Accel) |
| `config.go:120` | `BACKUP_STORAGE_PATH` (env-getrieben) ✔ |
| `hardware_service.go:96-97` | Mount-Prefix-Checks `/mnt/data`, `/mnt/backups` |
| `honeyfile_service.go:20` | `defaultHoneyfileRoot = "/mnt/data"` |
