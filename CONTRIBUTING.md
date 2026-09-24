# Mitwirken an NAS.AI

Diese Regeln gelten für **alle** Mitwirkenden, egal ob Mensch oder KI-Agent.
Sie werden auf GitHub automatisch geprüft. Ein PR, der sie verletzt, kann
nicht gemergt werden.

Technische Details zum Projekt stehen in
[`docs/development/DEV_GUIDE.md`](docs/development/DEV_GUIDE.md).

## Arbeitsweise

1. **Eine Aufgabe = ein Issue = ein Branch = ein PR.**
   Aufgaben werden als Issue mit der Vorlage *Task* angelegt (Ziel, Scope,
   Akzeptanzkriterien). Nur Dateien im Scope werden geändert.
2. **Branch** von aktuellem `main` erstellen:
   `feature/…`, `fix/…`, `docs/…`, `chore/…`, `ci/…`, `refactor/…`,
   `test/…`, `perf/…`, `hotfix/…`, `release/…` (Kleinbuchstaben, `-` statt Leerzeichen).
3. **Klein halten:** Ziel sind PRs unter 800 geänderten Zeilen. Größere
   Aufgaben in mehrere PRs aufteilen.
4. **Vor dem Push lokal prüfen** (im betroffenen Go-Modul):
   ```bash
   gofmt -l .          # muss leer sein
   go vet ./...
   go test -race ./...
   make security-scan  # in infrastructure/api
   ```
5. **PR gegen `main`** öffnen, Vorlage ausfüllen, mit `Closes #<issue>`
   auf das Issue verweisen.
6. **Nur der Admin mergt.** Niemand pusht direkt auf `main`, niemand
   mergt eigene PRs ohne Review des Code Owners.

## Commits

- **Autor ist immer die Adresse des Repository-Owners:**
  ```bash
  git config user.name  "Felix Freund"
  git config user.email "226513012+frnd1406@users.noreply.github.com"
  ```
- **Keine** `Co-Authored-By`-, `Claude-Session`- oder ähnlichen
  Zuschreibungszeilen.
- Englisch, [Conventional Commits](https://www.conventionalcommits.org/):
  `fix(api): reject empty upload path`, `docs: …`, `ci: …`.
  Dasselbe Format gilt für den PR-Titel.

## Parallel arbeiten (mehrere Agenten)

- Der Check **PR hygiene** kommentiert, wenn ein anderer offener PR
  dieselben Dateien ändert. Dann zuerst abstimmen oder nacheinander mergen.
- Vor dem Weiterarbeiten an einem älteren Branch `main` hineinmergen.
  Der Branch muss vor dem Merge aktuell sein.
- Nie auf fremde Branches pushen und nie History umschreiben (`rebase`,
  `--force`) auf Branches, die schon einen PR haben.
- PRs ohne Aktivität werden nach 14 Tagen als `stale` markiert und nach
  weiteren 7 Tagen geschlossen.

## Was nicht ohne Rückfrage geändert wird

- `.github/rulesets/`, `.github/repo-settings.json`, `.github/workflows/`
- Secrets, Tokens, Zertifikate: gehören **nie** ins Repo (auch nicht in
  Doku oder Tests). `gitleaks` blockiert den PR.
- `infrastructure/webui/`: derzeit pausiert.
- Agenten-Konfiguration (`.claude/`, `.codex/`, `.agents/`, `AGENTS.md`)
  bleibt lokal und wird nicht committet.

## Sicherheitslücken

Nicht als öffentliches Issue melden, sondern über
[Security Advisories](https://github.com/frnd1406/NAS-AI/security/advisories/new).
Siehe [`SECURITY.md`](SECURITY.md).
