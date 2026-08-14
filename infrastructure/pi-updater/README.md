# Pi Host-Updater (YubiKey-gated Patching)

Sichere, per YubiKey abgesicherte Patch-Steuerung für die Homelab-Pis (NAS + Firewall),
**ohne neuen Netzwerk-Dienst** — getriggert über den bereits gehärteten SSH-Kanal.
Vollständiges Design: [`docs/SPEICHER-ARCHITEKTUR.md`](../../docs/SPEICHER-ARCHITEKTUR.md) §12.

## Komponenten
- **`nas-updater.sh`** → nach `/usr/local/sbin/nas-updater.sh` (root, `0755`). Kann NUR:
  `status` · `upgrade` (apt full-upgrade) · `docker` (postgres+redis pull+recreate, nur NAS) · `reboot`.
  Kein Zugriff auf Vault/Daten.
- **`nas-updater-dispatch`** → nach `/usr/local/sbin/nas-updater-dispatch` (root, `0755`).
  Forced-Command-Wrapper: der dedizierte SSH-Key darf ausschließlich die vier Aktionen auslösen,
  **keine Shell**.

## Deploy (pro Pi)
```bash
sudo install -m 0755 -o root -g root nas-updater.sh          /usr/local/sbin/nas-updater.sh
sudo install -m 0755 -o root -g root nas-updater-dispatch.sh /usr/local/sbin/nas-updater-dispatch
```

## FIDO2-Key einrichten (einmalig, am Steuer-PC)
```powershell
# Windows-OpenSSH (v8.2+). Touch + PIN Pflicht.
& "C:\Windows\System32\OpenSSH\ssh-keygen.exe" -t ed25519-sk -O resident -O verify-required `
  -O application=ssh:nas-updater -f "$env:USERPROFILE\.ssh\id_ed25519_sk_updater" -C "nas-updater"
```
Public Key auf jedem Pi in `~/.ssh/authorized_keys` mit Forced-Command + Beschränkung:
```
restrict,verify-required,command="/usr/local/sbin/nas-updater-dispatch" sk-ssh-ed25519@openssh.com AAAA...== nas-updater
```

## Nutzung
```bash
ssh -i ~/.ssh/id_ed25519_sk_updater nasserver@192.168.1.81 status   # bzw. upgrade | docker | reboot
ssh -i ~/.ssh/id_ed25519_sk_updater firewall@192.168.1.82  status
```
Jede Aktion verlangt YubiKey-PIN + Touch (serverseitig via `verify-required` erzwungen).

## Sicherheitsprinzipien
- **Kein lauschender Dienst / kein neuer Port** — nur der gehärtete SSH-Kanal (key-only, ufw nur PC).
- Der Container-API-Prozess bleibt **unprivilegiert** und ruft den Updater **nie** auf.
- **Nie** für die MCP-/KI-Schicht erreichbar; **nur** auf ausdrückliche Nutzer-Aktion (Touch+PIN).
- `unattended-upgrades` deckt Debian-Security automatisch ab; der Updater ist v. a. für
  rpt-Kernel-Updates + Reboot und Docker-Images.
