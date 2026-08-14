// Custom hook for long-press detection with haptic feedback
import { useCallback, useRef } from 'react';

export function useLongPress(callback, options = {}) {
    const { delay = 500, onStart, onCancel } = options;
    const timeoutRef = useRef(null);
    const isLongPress = useRef(false);
    const targetRef = useRef(null);

    const start = useCallback((e) => {
        // Prevent context menu on long-press
        e.preventDefault();

        targetRef.current = e.target;
        isLongPress.current = false;
        onStart?.();

        timeoutRef.current = setTimeout(() => {
            isLongPress.current = true;

            // Trigger haptic feedback if available
            if (navigator.vibrate) {
                navigator.vibrate(50);
            }

            callback(e);
        }, delay);
    }, [callback, delay, onStart]);

    const stop = useCallback((e) => {
        if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
            timeoutRef.current = null;
        }

        if (!isLongPress.current) {
            onCancel?.();
        }

        targetRef.current = null;
    }, [onCancel]);

    const cancel = useCallback(() => {
        if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
            timeoutRef.current = null;
        }
        isLongPress.current = false;
        onCancel?.();
    }, [onCancel]);

    return {
        onTouchStart: start,
        onTouchEnd: stop,
        onTouchMove: cancel,
        onMouseDown: start,
        onMouseUp: stop,
        onMouseLeave: cancel,
        isLongPress,
    };
}
