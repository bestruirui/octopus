export type MorphingDialogLifecycleEvent = 'opened' | 'closed';

export function getMorphingDialogLifecycleEvent(
    previousIsOpen: boolean,
    nextIsOpen: boolean,
): MorphingDialogLifecycleEvent | null {
    if (!previousIsOpen && nextIsOpen) {
        return 'opened';
    }
    if (previousIsOpen && !nextIsOpen) {
        return 'closed';
    }
    return null;
}
