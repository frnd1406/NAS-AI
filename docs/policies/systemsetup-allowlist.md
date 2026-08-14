# Package Allowlist Management

## Allowlist Datei
- Speicherort: `/etc/nas/package-allowlist.yaml`
- Optionales Tracking (kein Blockierungsmechanismus).
- Struktur (Beispiel):
  ```yaml
  packages: []
  ```
- Versionierung via Git (`/etc/nas/.allowlist.git`).

### Vorab genehmigte Tool-Bundles
- **Baseline:** build-essential, git, vim, curl, wget, python3-{pip,venv,dev}, htop, iotop, nethogs, ncdu, net-tools, dnsutils, nmap, ufw, fail2ban, unattended-upgrades, logrotate, rsync, tar, gzip, unzip, jq.
- **Netzwerk & Security:** wireguard, qrencode, certbot, testssl.sh Release, iptables-persistent, amtool, nmap, ssl labs Freigabe.
- **Services & Storage:** samba, samba-common-bin, nfs-kernel-server, vsftpd, ffmpeg, openssl, rsync, cryptsetup, rclone, gnupg, zstd, smartmontools, hdparm, parted, mdadm, fio, lsscsi.
- **Container & Monitoring:** docker.io, docker-compose-plugin, docker-buildx-plugin (optional), trivy, grype, prometheus-node-exporter, nas_observability Stack (Prometheus, Alertmanager, Grafana), stress, stress-ng.
- **Development Stacks:** Node.js 18 LTS, npm/pnpm/yarn, @mui/material etc. (siehe WebUIAgent), poetry, fastapi, uvicorn, alembic, psycopg2-binary, redis, @nestjs/cli, typescript, ts-node, prisma, zod, pm2.
- **Mobile & CI/CD:** Xcode ≥15, SwiftLint, Alamofire, SwiftJWT, Realm, GRDB, Kingfisher, Lottie, fastlane, bundler, cocoapods, Firebase CLI.
- **Security Testing:** masscan, nmap, Subdomain-Enumeration (z.B. Sublist3r/Subfinder), httpx, whatweb, Nessus, OpenVAS, Nikto, OWASP ZAP, StackHawk CLI, Burp Suite (optional), testssl.sh, ssh-audit, docker-bench, lynis/CIS-CAT, trivy, grype.

## Prozess
1. Agent sendet `package_request` Event an Orchestrator:
   ```json
   {
     "agent": "WebUIAgent",
     "package": "nodejs",
     "version": "18.x",
     "source": "apt",
     "context": "Build Pipeline"
   }
   ```
2. Orchestrator loggt Anfrage und informiert SystemSetupAgent.
3. SystemSetupAgent prüft:
   - Quelle/Repo (Signatur, Maintainer, HTTPS)
   - CVE/Advisories (Debian Tracker, NVD)
   - Malware/Integrity (falls Drittquelle -> ClamAV/YARA)
4. Ergebnis `package_status` Event:
   ```json
   {
     "agent": "SystemSetupAgent",
     "package": "nodejs",
     "status": "approved",
     "notes": "LTS, CVEs geprüft bis 2024-01"
   }
   ```
5. Bei Freigabe: Eintrag in Allowlist optional (Dokumentation) und Installation ausführen (`apt-get install nodejs`).
6. Bei Ablehnung: Status `denied` + Begründung, optional Alternative anbieten.
7. Orchestrator aktualisiert Ticket, informiert anfragenden Agent.

## Automation
- Vollautomatischer Betrieb: Entscheidungen erfolgen ohne manuelle Zustimmung, basierend auf Policy und CVE-Prüfungen (Allowlist rein informativ).
- Script `package-approver.sh` (SystemSetupAgent):
  ```bash
  #!/bin/bash
  REQUEST_JSON=$1
  PACKAGE=$(jq -r '.package' "$REQUEST_JSON")
  SOURCE=$(jq -r '.source' "$REQUEST_JSON")
  # Prüfung durchführen...
  # Ergebnis nach stdout -> Orchestrator liest Antwort
  ```
- Cron/Service überwacht Event Queue (`/var/lib/orchestrator/package-queue/`) und schreibt Audit-Logs.
- Allowlist wird nach jeder Änderung signiert (`gpg --clearsign`) und per Git versioniert.
- `systemsetup-allowlist-review.timer` erinnert monatlich an eine manuelle Sichtprüfung (Logeintrag via `systemsetup-allowlist-review.service`).

## Audit
- Jede Entscheidung wird in `/var/log/package-approvals.log` festgehalten.
- Vierteljährliches Review der Allowlist.
- Review-Protokolle (Monats-Reminder aus dem Timer, Quartals-Review) im Statusdokument hinterlegen.
- Abgelaufene Pakete (EOL) markieren und Nachfolge planen.

## Notfall
- Im Incidentfall: Allowlist Freeze (`chmod 400`), nur Security-Team darf Ergänzungen vornehmen.
- Rollback: Git reflog nutzen (`git reset --hard <commit>`).
