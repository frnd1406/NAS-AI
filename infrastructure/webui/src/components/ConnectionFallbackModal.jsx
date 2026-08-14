import { useEffect, useState } from 'react';
import { WifiOff, ExternalLink, AlertTriangle } from 'lucide-react';
import { getCachedLocalIP, getLastSeenTimestamp } from '../lib/api';

/**
 * ConnectionFallbackModal - Shows when internet connection fails
 * 
 * This modal appears when:
 * 1. The browser detects it's offline, OR
 * 2. Multiple consecutive API requests fail with network errors
 * 
 * If a cached local IP is available, it offers a link to switch to local mode.
 */
export default function ConnectionFallbackModal() {
    const [showModal, setShowModal] = useState(false);
    const [localIP, setLocalIP] = useState(null);
    const [lastSeen, setLastSeen] = useState(null);

    useEffect(() => {
        const checkConnection = () => {
            // Get cached IP
            const ip = getCachedLocalIP();
            const ts = getLastSeenTimestamp();
            setLocalIP(ip);
            setLastSeen(ts);

            // Show modal if offline and we have a cached IP
            if (!navigator.onLine && ip) {
                setShowModal(true);
            }
        };

        // Initial check
        checkConnection();

        // Listen for offline/online events
        const handleOffline = () => {
            const ip = getCachedLocalIP();
            if (ip) {
                setLocalIP(ip);
                setLastSeen(getLastSeenTimestamp());
                setShowModal(true);
            }
        };

        const handleOnline = () => {
            setShowModal(false);
        };

        window.addEventListener('offline', handleOffline);
        window.addEventListener('online', handleOnline);

        return () => {
            window.removeEventListener('offline', handleOffline);
            window.removeEventListener('online', handleOnline);
        };
    }, []);

    // Allow programmatic triggering from outside (API error handler)
    useEffect(() => {
        const handleApiError = (event) => {
            if (event.detail?.type === 'network_error') {
                const ip = getCachedLocalIP();
                if (ip) {
                    setLocalIP(ip);
                    setLastSeen(getLastSeenTimestamp());
                    setShowModal(true);
                }
            }
        };

        window.addEventListener('nas:connection_error', handleApiError);
        return () => window.removeEventListener('nas:connection_error', handleApiError);
    }, []);

    const formatLastSeen = (timestamp) => {
        if (!timestamp) return 'Unbekannt';
        const date = new Date(timestamp);
        return date.toLocaleString('de-DE', {
            day: '2-digit',
            month: '2-digit',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
        });
    };

    const handleSwitchToLocal = () => {
        // Open local IP with HTTP (not HTTPS to avoid cert errors)
        window.open(`http://${localIP}:8080`, '_blank');
    };

    if (!showModal) return null;

    return (
        <div
            className="connection-fallback-overlay"
            style={{
                position: 'fixed',
                inset: 0,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: 'rgba(0, 0, 0, 0.5)',
                backdropFilter: 'blur(12px)',
                zIndex: 99999,
            }}
        >
            <div
                className="connection-fallback-card"
                style={{
                    maxWidth: '460px',
                    width: '90%',
                    padding: '28px 32px',
                    borderRadius: '20px',
                    background: 'linear-gradient(135deg, rgba(251, 146, 60, 0.25), rgba(194, 65, 12, 0.35))',
                    border: '1px solid rgba(251, 146, 60, 0.55)',
                    boxShadow: '0 20px 70px rgba(251, 146, 60, 0.35)',
                    color: '#fff',
                    backdropFilter: 'blur(18px)',
                    fontFamily: "'Inter', system-ui, -apple-system, sans-serif",
                }}
            >
                {/* Warning Badge */}
                <div
                    style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: '8px',
                        padding: '6px 14px',
                        borderRadius: '999px',
                        background: 'rgba(251, 146, 60, 0.28)',
                        border: '1px solid rgba(251, 146, 60, 0.55)',
                        textTransform: 'uppercase',
                        letterSpacing: '0.08em',
                        fontSize: '12px',
                        fontWeight: 700,
                    }}
                >
                    <WifiOff size={14} />
                    Verbindung unterbrochen
                </div>

                {/* Title */}
                <h2
                    style={{
                        margin: '16px 0 8px 0',
                        fontSize: '24px',
                        fontWeight: 800,
                        letterSpacing: '0.01em',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '10px',
                    }}
                >
                    <AlertTriangle size={24} />
                    Internet nicht erreichbar
                </h2>

                {/* Body */}
                <p
                    style={{
                        margin: '0 0 16px 0',
                        color: 'rgba(255, 255, 255, 0.9)',
                        lineHeight: 1.5,
                        fontSize: '15px',
                    }}
                >
                    Der Server ist über das Internet nicht erreichbar. Wenn du dich im lokalen Netzwerk befindest,
                    kannst du direkt auf den Server zugreifen.
                </p>

                {/* Local IP info */}
                {localIP && (
                    <div
                        style={{
                            padding: '12px 16px',
                            borderRadius: '12px',
                            background: 'rgba(255, 255, 255, 0.1)',
                            marginBottom: '16px',
                            fontSize: '14px',
                        }}
                    >
                        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}>
                            <span style={{ color: 'rgba(255, 255, 255, 0.7)' }}>Lokale Adresse:</span>
                            <span style={{ fontWeight: 600, fontFamily: 'monospace' }}>{localIP}</span>
                        </div>
                        <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                            <span style={{ color: 'rgba(255, 255, 255, 0.7)' }}>Zuletzt gesehen:</span>
                            <span style={{ color: 'rgba(255, 255, 255, 0.85)' }}>{formatLastSeen(lastSeen)}</span>
                        </div>
                    </div>
                )}

                {/* Action Button */}
                <button
                    onClick={handleSwitchToLocal}
                    style={{
                        width: '100%',
                        padding: '14px 20px',
                        borderRadius: '12px',
                        border: 'none',
                        background: 'linear-gradient(135deg, #f97316, #ea580c)',
                        color: '#fff',
                        fontSize: '16px',
                        fontWeight: 700,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        gap: '10px',
                        transition: 'transform 0.15s ease, box-shadow 0.15s ease',
                    }}
                    onMouseEnter={(e) => {
                        e.target.style.transform = 'translateY(-2px)';
                        e.target.style.boxShadow = '0 8px 25px rgba(249, 115, 22, 0.4)';
                    }}
                    onMouseLeave={(e) => {
                        e.target.style.transform = 'translateY(0)';
                        e.target.style.boxShadow = 'none';
                    }}
                >
                    <ExternalLink size={18} />
                    Zum lokalen Modus wechseln
                </button>

                {/* Hint */}
                <p
                    style={{
                        margin: '12px 0 0 0',
                        color: 'rgba(255, 255, 255, 0.6)',
                        fontSize: '12px',
                        textAlign: 'center',
                    }}
                >
                    Hinweis: Du musst dich im selben Netzwerk wie der Server befinden.
                </p>
            </div>
        </div>
    );
}
