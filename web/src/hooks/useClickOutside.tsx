import { RefObject, useEffect } from 'react';

function useClickOutside<T extends HTMLElement>(
  ref: RefObject<T>,
  handler: (event: PointerEvent | TouchEvent) => void,
  shouldIgnore?: (event: PointerEvent | TouchEvent) => boolean
): void {
  useEffect(() => {
    const handleClickOutside = (event: PointerEvent | TouchEvent) => {
      if (shouldIgnore?.(event)) {
        return;
      }
      if (!ref || !ref.current || ref.current.contains(event.target as Node)) {
        return;
      }

      handler(event);
    };

    document.addEventListener('pointerdown', handleClickOutside, true);
    document.addEventListener('touchstart', handleClickOutside, true);

    return () => {
      document.removeEventListener('pointerdown', handleClickOutside, true);
      document.removeEventListener('touchstart', handleClickOutside, true);
    };
  }, [ref, handler, shouldIgnore]);
}

export default useClickOutside;
