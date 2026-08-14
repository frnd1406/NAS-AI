// Custom hook for file selection (multi-select feature)

import { useState, useCallback, useMemo } from 'react';

export function useFileSelection(files) {
    const [selectedPaths, setSelectedPaths] = useState(new Set());

    // Select a single file (toggle)
    const toggleSelect = useCallback((filePath) => {
        setSelectedPaths(prev => {
            const newSet = new Set(prev);
            if (newSet.has(filePath)) {
                newSet.delete(filePath);
            } else {
                newSet.add(filePath);
            }
            return newSet;
        });
    }, []);

    // Select all files
    const selectAll = useCallback((allPaths) => {
        setSelectedPaths(new Set(allPaths));
    }, []);

    // Clear all selections
    const clearSelection = useCallback(() => {
        setSelectedPaths(new Set());
    }, []);

    // Toggle select all / clear all
    const toggleSelectAll = useCallback((allPaths) => {
        if (selectedPaths.size === allPaths.length && allPaths.length > 0) {
            clearSelection();
        } else {
            selectAll(allPaths);
        }
    }, [selectedPaths.size, selectAll, clearSelection]);

    // Check if a path is selected
    const isSelected = useCallback((filePath) => {
        return selectedPaths.has(filePath);
    }, [selectedPaths]);

    // Check if all are selected
    const allSelected = useMemo(() => {
        if (!files || files.length === 0) return false;
        return files.every(f => selectedPaths.has(f.name));
    }, [files, selectedPaths]);

    // Get selected items
    const selectedItems = useMemo(() => {
        if (!files) return [];
        return files.filter(f => selectedPaths.has(f.name));
    }, [files, selectedPaths]);

    // Get count of selected items
    const selectedCount = selectedPaths.size;

    return {
        selectedPaths,
        selectedItems,
        selectedCount,
        allSelected,
        toggleSelect,
        selectAll,
        clearSelection,
        toggleSelectAll,
        isSelected,
    };
}
