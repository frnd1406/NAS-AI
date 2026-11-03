# smb-monitor

Reports UFW blocks on port 445 and failed SMB logins to `POST /api/v1/system/events`.
Disabled by default.

## Install (on the NAS host, as root)
```bash
install -m 700 smb-monitor.sh /home/nasserver/bin/smb-monitor.sh
install -m 644 smb-monitor.service smb-monitor.timer /etc/systemd/system/
install -m 600 /dev/null /etc/default/smb-monitor
printf 'SMB_MONITOR_ENABLED=false\nMONITORING_TOKEN_FILE=<path to existing secrets file>\n' > /etc/default/smb-monitor
systemctl daemon-reload
```
Deploy the new `nas-api` image and wait until it is healthy first, then set `SMB_MONITOR_ENABLED=true` and:
```bash
systemctl enable --now smb-monitor.timer
```
## Rollback
```bash
systemctl disable --now smb-monitor.timer
rm /etc/systemd/system/smb-monitor.{service,timer} /home/nasserver/bin/smb-monitor.sh
```
Cursors are kept in `/var/lib/smb-monitor/` (`cursor` = kernel log, `cursor.smbd`).
The smbd log patterns (`NT_STATUS_WRONG_PASSWORD`, ...) should be checked against the real `journalctl -u smbd` output of your Samba version.
