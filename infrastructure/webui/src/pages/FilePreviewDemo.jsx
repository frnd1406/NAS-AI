import { useState } from 'react';
import {
  FileText,
  Sparkles,
  Search,
  MessageSquare,
  Code,
  Image as ImageIcon,
  FileJson,
  AlertCircle
} from 'lucide-react';
import { FilePreviewPanel } from '../components/Files/FilePreviewPanel';
import { FileSelector } from '../components/Files/FileSelector';

/**
 * FilePreviewDemo - Complete demo page for the File Preview System
 * Demonstrates:
 * - Single file auto-open
 * - Multiple file selection
 * - Different file types (txt, json, code, images)
 * - Similarity scoring
 * - Interactive file navigation
 */
export default function FilePreviewDemo() {
  const [activeDemo, setActiveDemo] = useState(null);
  const [previewFiles, setPreviewFiles] = useState([]);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [showPreview, setShowPreview] = useState(false);
  const [showSelector, setShowSelector] = useState(false);

  // Demo scenarios
  const demoScenarios = [
    {
      id: 'single-file',
      title: 'Single File Demo',
      description: 'Zeigt eine einzelne Datei automatisch an',
      icon: FileText,
      color: 'blue',
      files: [
        {
          file_id: 'steuerbescheid_2024.txt',
          file_path: '/mnt/data/finanzen/steuerbescheid_2024.txt',
          similarity: 0.95,
          content: `FINANZAMT BERLIN
Steuernummer: 123/4567/89012

EINKOMMENSTEUERBESCHEID 2024

Steuerpflichtiger: Max Mustermann
Anschrift: Hauptstraße 123, 10115 Berlin

Ermittlung des zu versteuernden Einkommens:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Einkünfte aus nichtselbständiger Arbeit:    65.000,00 EUR
Werbungskosten (Pauschbetrag):              -1.200,00 EUR
Sonderausgaben:                             -2.500,00 EUR
Vorsorgeaufwendungen:                       -5.000,00 EUR
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Zu versteuerndes Einkommen:                 56.300,00 EUR

Festgesetzte Einkommensteuer:               15.800,00 EUR
Solidaritätszuschlag:                          869,00 EUR
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Gesamtbetrag:                               16.669,00 EUR

Bereits gezahlte Lohnsteuer:                17.200,00 EUR

Erstattung: 531,00 EUR

Zahlungsfrist: 30.06.2024

Rechtsbehelfsbelehrung:
Gegen diesen Bescheid kann innerhalb eines Monats nach Bekanntgabe
Einspruch eingelegt werden.

gez. Sachbearbeiter/in
Finanzamt Berlin`
        }
      ]
    },
    {
      id: 'multiple-files',
      title: 'Multiple Files Demo',
      description: 'Nutzer wählt aus mehreren relevanten Dateien',
      icon: Search,
      color: 'emerald',
      files: [
        {
          file_id: 'rechnung_amazon_2024.txt',
          file_path: '/mnt/data/finanzen/rechnungen/rechnung_amazon_2024.txt',
          similarity: 0.92,
          content: `AMAZON DEUTSCHLAND
Kundenservice | Rechnungsabteilung
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

RECHNUNG
Rechnungsnummer: RE-2024567
Rechnungsdatum: 15.03.2024
Fälligkeitsdatum: 29.03.2024

Rechnungsempfänger:
Max Mustermann
Hauptstraße 123, 10115 Berlin

LEISTUNGSÜBERSICHT:
───────────────────────────────────────────────────────────────
  Software-Lizenz Premium Paket                     299.99 EUR
  Cloud-Speicher 1TB monatlich                       29.99 EUR
───────────────────────────────────────────────────────────────
  Zwischensumme (netto):                            329.98 EUR
  MwSt. 19%:                                         62.70 EUR
  ═══════════════════════════════════════════════════════════════
  GESAMTBETRAG:                                     392.68 EUR

Zahlungsart: Kreditkarte
Vielen Dank für Ihr Vertrauen!
Amazon Deutschland`
        },
        {
          file_id: 'krypto_binance_2024.txt',
          file_path: '/mnt/data/finanzen/krypto/krypto_binance_2024.txt',
          similarity: 0.78,
          content: `BINANCE - KONTOAUSZUG KRYPTOWÄHRUNGEN
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Kontoinhaber: Max Mustermann
Kunden-ID: 789456
Zeitraum: 03/2024

HANDELSÜBERSICHT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Kryptowährung        | Menge    | Kaufkurs        | Verkaufskurs    | Gewinn/Verlust
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Bitcoin (BTC)        | 0.1500   | Kauf: 45000€    | Verkauf: 52000€ | +1050.00€
Ethereum (ETH)       | 2.5000   | Kauf: 2800€     | Verkauf: 3200€  | +1000.00€
Solana (SOL)         | 50.0000  | Kauf: 90€       | Verkauf: 110€   | +1000.00€
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
GESAMTERGEBNIS: +3050.00 EUR

HINWEIS ZUR STEUERPFLICHT:
Kryptowährungsgewinne sind nach § 23 EStG steuerpflichtig.
Spekulationsfrist: Gewinne nach 1 Jahr Haltedauer sind steuerfrei.

Binance GmbH - Kryptobörse Deutschland`
        },
        {
          file_id: 'kontoauszug_sparkasse_2024.txt',
          file_path: '/mnt/data/finanzen/kontoauszuege/kontoauszug_sparkasse_2024.txt',
          similarity: 0.65,
          content: `SPARKASSE BERLIN
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

KONTOAUSZUG
Kontonummer: DE89 3704 0044 0532 0130 00
Kontoinhaber: Max Mustermann
Auszugszeitraum: 03/2024

Anfangssaldo: 8.500,00 EUR

UMSÄTZE:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Datum      | Buchungstext                         | Betrag         | Saldo
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
01.03.2024 | Gehalt SAP SE                        | +4500.00 EUR   | 13000.00 EUR
05.03.2024 | REWE Supermarkt                      | -85.50 EUR     | 12914.50 EUR
10.03.2024 | Amazon EU                            | -392.68 EUR    | 12521.82 EUR
15.03.2024 | Netflix                              | -12.99 EUR     | 12508.83 EUR
20.03.2024 | Miete Wohnung                        | -850.00 EUR    | 11658.83 EUR
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
ENDSALDO: 11.658,83 EUR

Sparkasse Berlin | IBAN: DE89 3704 0044 0532 0130 00`
        }
      ]
    },
    {
      id: 'json-demo',
      title: 'JSON File Demo',
      description: 'Zeigt strukturierte JSON-Daten an',
      icon: FileJson,
      color: 'violet',
      files: [
        {
          file_id: 'api_config.json',
          file_path: '/mnt/data/config/api_config.json',
          similarity: 0.88,
          content: JSON.stringify({
            "api": {
              "version": "v1.0.0",
              "endpoints": {
                "search": "/api/v1/search",
                "ask": "/api/v1/ask",
                "files": "/api/v1/files"
              },
              "settings": {
                "embedding_model": "mxbai-embed-large",
                "llm_model": "llama3.2",
                "embedding_dim": 1024,
                "max_results": 10
              }
            },
            "database": {
              "host": "postgres",
              "port": 5432,
              "name": "nas_db",
              "pool_size": 10
            },
            "features": {
              "semantic_search": true,
              "rag_enabled": true,
              "file_preview": true,
              "auto_indexing": true
            }
          }, null, 2)
        }
      ]
    },
    {
      id: 'code-demo',
      title: 'Code File Demo',
      description: 'Zeigt Quellcode mit Syntax-Highlighting',
      icon: Code,
      color: 'amber',
      files: [
        {
          file_id: 'search_handler.go',
          file_path: '/mnt/data/code/search_handler.go',
          similarity: 0.91,
          content: `package handlers

import (
    "context"
    "encoding/json"
    "net/http"
    "github.com/gin-gonic/gin"
)

type SearchResult struct {
    FilePath   string  \`json:"file_path"\`
    Content    string  \`json:"content"\`
    Similarity float64 \`json:"similarity"\`
}

// SearchHandler handles semantic search requests
func SearchHandler(db *database.DB, aiServiceURL string) gin.HandlerFunc {
    return func(c *gin.Context) {
        query := c.Query("q")
        if query == "" {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "missing query parameter",
            })
            return
        }

        // Get embedding from AI agent
        embedding, err := fetchEmbedding(ctx, query)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "embedding failed",
            })
            return
        }

        // Perform vector search
        results := performVectorSearch(db, embedding)

        c.JSON(http.StatusOK, gin.H{
            "query": query,
            "results": results,
        })
    }
}`
        }
      ]
    }
  ];

  const handleDemoClick = (demo) => {
    setActiveDemo(demo.id);
    setPreviewFiles(demo.files);
    setCurrentIndex(0);

    if (demo.files.length === 1) {
      // Auto-open single file
      setShowSelector(false);
      setShowPreview(true);
    } else {
      // Show file selector for multiple files
      setShowSelector(true);
      setShowPreview(false);
    }
  };

  const handleFileSelect = (index) => {
    setCurrentIndex(index);
    setShowSelector(false);
    setShowPreview(true);
  };

  return (
    <div className="min-h-screen bg-[#0a0a0c] text-slate-200 p-8">
      {/* Header */}
      <div className="max-w-7xl mx-auto mb-12">
        <div className="flex items-center gap-3 mb-4">
          <div className="p-3 bg-gradient-to-br from-blue-500 to-violet-600 rounded-xl">
            <Sparkles size={32} className="text-white" />
          </div>
          <div>
            <h1 className="text-4xl font-bold text-white">File Preview System</h1>
            <p className="text-slate-400 mt-1">Interaktive Dateiauswahl mit intelligenter Vorschau</p>
          </div>
        </div>

        {/* Feature Highlights */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mt-8">
          {[
            { icon: Search, label: 'Semantic Search', desc: 'AI-gestützte Suche' },
            { icon: FileText, label: 'Multi-Format', desc: 'Alle Dateitypen' },
            { icon: MessageSquare, label: 'Smart Selection', desc: 'Intelligente Auswahl' },
            { icon: AlertCircle, label: 'Relevance Score', desc: 'Similarity-basiert' }
          ].map((feature, i) => (
            <div key={i} className="p-4 bg-slate-800/50 border border-white/10 rounded-xl">
              <feature.icon size={24} className="text-blue-400 mb-2" />
              <h3 className="text-white font-semibold text-sm">{feature.label}</h3>
              <p className="text-slate-400 text-xs mt-1">{feature.desc}</p>
            </div>
          ))}
        </div>
      </div>

      {/* Demo Scenarios */}
      <div className="max-w-7xl mx-auto">
        <h2 className="text-2xl font-bold text-white mb-6">Demo Szenarien</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-12">
          {demoScenarios.map((demo) => {
            const Icon = demo.icon;
            const isActive = activeDemo === demo.id;

            return (
              <button
                key={demo.id}
                onClick={() => handleDemoClick(demo)}
                className={`group relative p-6 rounded-2xl border-2 transition-all duration-300 text-left ${isActive
                    ? `border-${demo.color}-500 bg-${demo.color}-500/10 shadow-lg shadow-${demo.color}-500/20`
                    : 'border-white/10 bg-slate-800/30 hover:border-white/20 hover:bg-slate-800/50'
                  }`}
              >
                <div className={`p-3 bg-${demo.color}-500/20 rounded-xl inline-block mb-4 group-hover:scale-110 transition-transform`}>
                  <Icon size={28} className={`text-${demo.color}-400`} />
                </div>
                <h3 className="text-white font-bold text-lg mb-2">{demo.title}</h3>
                <p className="text-slate-400 text-sm mb-3">{demo.description}</p>
                <div className="flex items-center gap-2 text-xs">
                  <span className={`px-2 py-1 bg-${demo.color}-500/20 text-${demo.color}-400 rounded`}>
                    {demo.files.length} {demo.files.length === 1 ? 'Datei' : 'Dateien'}
                  </span>
                  {isActive && (
                    <span className="px-2 py-1 bg-emerald-500/20 text-emerald-400 rounded animate-pulse">
                      Aktiv
                    </span>
                  )}
                </div>
              </button>
            );
          })}
        </div>

        {/* File Selector (when multiple files) */}
        {showSelector && previewFiles.length > 1 && (
          <div className="max-w-3xl mx-auto mb-8">
            <div className="bg-slate-800/50 border border-white/10 rounded-2xl p-6">
              <FileSelector
                files={previewFiles}
                onSelectFile={handleFileSelect}
                autoSelectSingle={false}
              />
            </div>
          </div>
        )}

        {/* Instructions */}
        <div className="max-w-3xl mx-auto bg-blue-500/10 border border-blue-500/30 rounded-xl p-6">
          <div className="flex items-start gap-3">
            <AlertCircle size={24} className="text-blue-400 flex-shrink-0 mt-1" />
            <div>
              <h3 className="text-white font-semibold mb-2">So funktioniert es:</h3>
              <ul className="text-slate-300 text-sm space-y-2">
                <li>1. Wähle ein Demo-Szenario oben aus</li>
                <li>2. Bei einer Datei: Automatische Vorschau rechts</li>
                <li>3. Bei mehreren Dateien: Wähle die relevanteste aus</li>
                <li>4. Navigation zwischen Dateien mit Pfeiltasten</li>
                <li>5. Download und externe Links verfügbar</li>
              </ul>
            </div>
          </div>
        </div>
      </div>

      {/* File Preview Panel */}
      {showPreview && previewFiles.length > 0 && (
        <FilePreviewPanel
          files={previewFiles}
          currentIndex={currentIndex}
          onClose={() => {
            setShowPreview(false);
            setActiveDemo(null);
          }}
          onNavigate={setCurrentIndex}
        />
      )}
    </div>
  );
}
