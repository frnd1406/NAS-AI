import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { apiRequest } from '../lib/api';

const VaultContext = createContext();

export function VaultProvider({ children }) {
    const [isUnlocked, setIsUnlocked] = useState(false);
    const [password, setPassword] = useState(null); // Store password for uploads (Zero-Knowledge: RAM only)
    const [hasVault, setHasVault] = useState(false);
    const [loading, setLoading] = useState(true);

    const checkVaultStatus = useCallback(async () => {
        try {
            const status = await apiRequest('/api/v1/vault/status');
            setHasVault(status.configured);
            setIsUnlocked(!status.locked);
            if (status.locked) {
                setPassword(null); // Clear password if locked on server
            }
        } catch (err) {
            console.error('Failed to check vault status:', err);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        checkVaultStatus();

        // Listen for global lock events (e.g., from 423 responses)
        const handleLockEvent = () => {
            setIsUnlocked(false);
            setPassword(null);
            checkVaultStatus();
        };

        window.addEventListener('vault-locked', handleLockEvent);
        return () => window.removeEventListener('vault-locked', handleLockEvent);
    }, [checkVaultStatus]);

    const unlock = async (pwd) => {
        try {
            await apiRequest('/api/v1/system/vault/unlock', {
                method: 'POST',
                body: JSON.stringify({ password: pwd })
            });
            setIsUnlocked(true);
            setPassword(pwd); // Store in RAM for file operations
            return true;
        } catch (err) {
            console.error('Unlock failed:', err);
            throw err;
        }
    };

    const setup = async (pwd) => {
        try {
            await apiRequest('/api/v1/system/vault/setup', {
                method: 'POST',
                body: JSON.stringify({ masterPassword: pwd })
            });
            setHasVault(true);
            setIsUnlocked(true);
            setPassword(pwd); // Store in RAM for file operations
            return true;
        } catch (err) {
            console.error('Setup failed:', err);
            throw err;
        }
    };

    const lock = async () => {
        try {
            await apiRequest('/api/v1/system/vault/lock', { method: 'POST' });
        } catch (err) {
            console.warn('Lock failed on server (might already be locked):', err);
        } finally {
            setIsUnlocked(false);
            setPassword(null); // CLEAR PASSWORD FROM MEMORY
        }
    };

    return (
        <VaultContext.Provider value={{
            isUnlocked,
            password, // Exposed for useFileStorage
            hasVault,
            loading,
            unlock,
            setup,
            lock,
            checkVaultStatus
        }}>
            {children}
        </VaultContext.Provider>
    );
}

export function useVault() {
    return useContext(VaultContext);
}
