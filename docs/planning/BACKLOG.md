# 📋 NAS Server - Feature Backlog

> Zuletzt aktualisiert: 2024-12-07

## ✅ Abgeschlossen

### Files Manager Refactoring (2024-12-07)
- [x] Files.jsx von 1134 auf ~300 Zeilen reduziert
- [x] Components extrahiert: FileCard, FileListView, FileGridView, FileToolbar, TrashView, etc.
- [x] Hooks extrahiert: useFileStorage, useFilePreview, useDragAndDrop, useFileSelection
- [x] Utils extrahiert: fileUtils.js

### Files Manager Features (2024-12-07)
- [x] Suchfilter (Instant-Filter nach Dateinamen)
- [x] Mehrfachauswahl mit Checkboxen
- [x] Batch-Download als ZIP
- [x] Batch-Delete
- [x] Ordner als ZIP herunterladen
- [x] Backend: `/download-zip`, `/batch-download`, `/mkdir` Endpoints

---

## 🔜 Geplant / Ideen-Parkplatz

### Priorität HOCH
_Aktuell keine offenen Punkte_

### Priorität MITTEL
| Feature | Beschreibung | Aufwand |
|---------|-------------|---------|
| ZIP-Entpacken | Upload ZIP → automatisch entpacken | Hoch (Security!) |
| Drag & Drop Verschieben | Dateien in Ordner ziehen | Mittel |

### Priorität NIEDRIG
| Feature | Beschreibung | Aufwand |
|---------|-------------|---------|
| Streaming-ZIP | Für große Ordner (>500MB) | Hoch |
| Sortierung | Nach Name/Größe/Datum sortieren | Niedrig |
| Kontextmenü | Rechtsklick-Aktionen | Mittel |
| Datei-Versionshistorie | Alte Versionen behalten | Hoch |

---

## 🐛 Bekannte Issues
_Aktuell keine bekannten Issues_

---

## 📝 Notizen

### Architektur-Entscheidungen
- **GlassCard** als wiederverwendbare UI-Komponente
- **Custom Hooks** für Business-Logik (useFileStorage, useFilePreview)
- **Utility-Functions** in `/utils/` für reine Funktionen

### Wichtige Dateien
```
infrastructure/webui/src/
├── pages/Files.jsx              # Hauptkomponente
├── hooks/
│   ├── useFileStorage.js        # CRUD-Operationen
│   ├── useFilePreview.js        # Preview-Modal
│   ├── useFileSelection.js      # Mehrfachauswahl
│   └── useDragAndDrop.js        # Drag & Drop
├── components/
│   ├── FileCard.jsx             # Grid-Karte
│   ├── FileListView.jsx         # Tabellen-Ansicht
│   ├── FileGridView.jsx         # Grid-Ansicht
│   ├── FileToolbar.jsx          # Toolbar mit Actions
│   └── ui/GlassCard.jsx         # Glassmorphism-Card
└── utils/fileUtils.js           # Utility-Funktionen

infrastructure/api/src/
├── handlers/storage.go          # Storage-Handler
└── services/storage_service.go  # Storage-Service
```
