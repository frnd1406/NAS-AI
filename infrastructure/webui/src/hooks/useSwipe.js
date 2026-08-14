// Custom hook for swipe gesture detection
import { useRef, useCallback } from 'react';

export function useSwipe(onSwipeLeft, onSwipeRight, { threshold = 50 } = {}) {
    const touchStartX = useRef(0);
    const touchStartY = useRef(0);
    const isSwipe = useRef(false);

    const handleTouchStart = useCallback((e) => {
        touchStartX.current = e.touches[0].clientX;
        touchStartY.current = e.touches[0].clientY;
        isSwipe.current = false;
    }, []);

    const handleTouchMove = useCallback((e) => {
        if (!touchStartX.current) return;

        const xDiff = touchStartX.current - e.touches[0].clientX;
        const yDiff = Math.abs(touchStartY.current - e.touches[0].clientY);

        // Only consider horizontal swipes (more horizontal than vertical)
        if (Math.abs(xDiff) > yDiff && Math.abs(xDiff) > threshold) {
            isSwipe.current = true;
        }
    }, [threshold]);

    const handleTouchEnd = useCallback((e) => {
        if (!isSwipe.current) return;

        const xDiff = touchStartX.current - e.changedTouches[0].clientX;

        if (xDiff > threshold) {
            // Swiped left
            onSwipeLeft?.();
        } else if (xDiff < -threshold) {
            // Swiped right
            onSwipeRight?.();
        }

        touchStartX.current = 0;
        touchStartY.current = 0;
        isSwipe.current = false;
    }, [threshold, onSwipeLeft, onSwipeRight]);

    return {
        onTouchStart: handleTouchStart,
        onTouchMove: handleTouchMove,
        onTouchEnd: handleTouchEnd,
    };
}
