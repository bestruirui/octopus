'use client';

import {
    type ReactNode,
    useCallback,
    useEffect,
    useMemo,
    useRef,
    useState,
} from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';

const BREAKPOINTS = {
    sm: 640,
    md: 768,
    lg: 960,
    xl: 1280,
    '2xl': 1536,
} as const;

type Breakpoint = keyof typeof BREAKPOINTS;
type ResponsiveColumns = Partial<Record<Breakpoint | 'default', number>>;

interface VirtualizedGridProps<T> {
    items: T[];
    layout?: 'grid' | 'list';
    columns: ResponsiveColumns;
    estimateItemHeight: number;
    gap?: number;
    overscan?: number;
    getItemKey: (item: T, index: number) => string | number;
    renderItem: (item: T, index: number) => ReactNode;
    footer?: ReactNode;
    onReachEnd?: () => void;
    reachEndEnabled?: boolean;
    reachEndOffset?: number;
}

function getColumnsForWidth(
    width: number,
    columns: ResponsiveColumns,
): number {
    if (width >= BREAKPOINTS['2xl'] && columns['2xl'] !== undefined) return columns['2xl'];
    if (width >= BREAKPOINTS.xl && columns.xl !== undefined) return columns.xl;
    if (width >= BREAKPOINTS.lg && columns.lg !== undefined) return columns.lg;
    if (width >= BREAKPOINTS.md && columns.md !== undefined) return columns.md;
    if (width >= BREAKPOINTS.sm && columns.sm !== undefined) return columns.sm;
    return columns.default ?? 1;
}

export function VirtualizedGrid<T>({
    items,
    layout = 'grid',
    columns,
    estimateItemHeight,
    gap = 16,
    overscan = 4,
    getItemKey,
    renderItem,
    footer = null,
    onReachEnd,
    reachEndEnabled = false,
    reachEndOffset = 1,
}: VirtualizedGridProps<T>) {
    'use no memo';

    const [containerWidth, setContainerWidth] = useState(() =>
        typeof window === 'undefined' ? 1024 : window.innerWidth
    );
    const containerRef = useRef<HTMLDivElement | null>(null);
    const reachEndTriggeredRef = useRef(false);

    useEffect(() => {
        const el = containerRef.current;
        if (!el) return;

        const updateWidth = () => {
            const nextWidth = el.clientWidth;
            setContainerWidth((prev) => (prev === nextWidth ? prev : nextWidth));
        };

        updateWidth();

        if (typeof ResizeObserver === 'undefined') return;
        const observer = new ResizeObserver(updateWidth);
        observer.observe(el);

        return () => {
            observer.disconnect();
        };
    }, []);

    const columnCount = useMemo(() => {
        if (layout === 'list') return 1;
        return Math.max(1, getColumnsForWidth(containerWidth, columns));
    }, [layout, containerWidth, columns]);

    const itemRowCount = useMemo(
        () => (items.length === 0 ? 0 : Math.ceil(items.length / columnCount)),
        [items.length, columnCount]
    );
    const hasFooterRow = footer !== null;
    const rowCount = itemRowCount + (hasFooterRow ? 1 : 0);
    const getVirtualRowKey = useCallback((rowIndex: number) => {
        if (hasFooterRow && rowIndex === itemRowCount) {
            return '__virtual-footer__';
        }

        const rowStartIndex = rowIndex * columnCount;
        const firstItem = items[rowStartIndex];
        if (!firstItem) return `row-empty-${rowIndex}`;

        // Keep row keys stable across prepend/append updates (especially log stream updates),
        // otherwise virtualizer measurements are constantly invalidated and spacing falls back to estimates.
        return `row-${String(getItemKey(firstItem, rowStartIndex))}`;
    }, [hasFooterRow, itemRowCount, columnCount, items, getItemKey]);

    // eslint-disable-next-line react-hooks/incompatible-library
    const rowVirtualizer = useVirtualizer({
        count: rowCount,
        getScrollElement: () => containerRef.current,
        getItemKey: getVirtualRowKey,
        estimateSize: () => estimateItemHeight + gap,
        // Use layout height (not transformed visual height) to avoid scale-animation
        // shrinking measurements during page enter transitions.
        measureElement: (element) =>
            element instanceof HTMLElement
                ? element.offsetHeight
                : element.getBoundingClientRect().height,
        overscan,
    });

    const virtualRows = rowVirtualizer.getVirtualItems();

    useEffect(() => {
        if (!onReachEnd || !reachEndEnabled || itemRowCount === 0) return;

        const lastVirtualIndex = virtualRows.length > 0 ? virtualRows[virtualRows.length - 1]!.index : -1;
        const triggerIndex = Math.max(0, itemRowCount - 1 - reachEndOffset);
        if (lastVirtualIndex < triggerIndex) {
            reachEndTriggeredRef.current = false;
            return;
        }
        if (reachEndTriggeredRef.current) return;

        reachEndTriggeredRef.current = true;
        onReachEnd();
    }, [onReachEnd, reachEndEnabled, itemRowCount, reachEndOffset, virtualRows]);

    return (
        <div className="relative h-full min-h-0 w-full">
            <div
                ref={containerRef}
                className="relative h-full w-full overflow-y-auto overscroll-contain rounded-t-3xl"
            >
                {rowCount === 0 ? null : (
                    <div className="relative w-full" style={{ height: `${rowVirtualizer.getTotalSize()}px` }}>
                        {virtualRows.map((virtualRow) => {
                            if (hasFooterRow && virtualRow.index === itemRowCount) {
                                return (
                                    <div
                                        key={virtualRow.key}
                                        data-index={virtualRow.index}
                                        ref={rowVirtualizer.measureElement}
                                        className="absolute left-0 top-0 w-full"
                                        style={{
                                            transform: `translateY(${virtualRow.start}px)`,
                                        }}
                                    >
                                        {footer}
                                    </div>
                                );
                            }

                            const rowStartIndex = virtualRow.index * columnCount;
                            const rowEndIndex = Math.min(rowStartIndex + columnCount, items.length);
                            const rowItems = items.slice(rowStartIndex, rowEndIndex);
                            const rowPaddingBottom = virtualRow.index === itemRowCount - 1 && !hasFooterRow ? 0 : gap;

                            return (
                                <div
                                    key={virtualRow.key}
                                    data-index={virtualRow.index}
                                    ref={rowVirtualizer.measureElement}
                                    className="absolute left-0 top-0 w-full"
                                    style={{
                                        transform: `translateY(${virtualRow.start}px)`,
                                    }}
                                >
                                    <div
                                        className="grid"
                                        style={{
                                            gridTemplateColumns: `repeat(${columnCount}, minmax(0, 1fr))`,
                                            gap: `${gap}px`,
                                            paddingBottom: `${rowPaddingBottom}px`,
                                        }}
                                    >
                                        {rowItems.map((item, columnIndex) => {
                                            const itemIndex = rowStartIndex + columnIndex;
                                            return (
                                                <div key={String(getItemKey(item, itemIndex))} className="min-w-0">
                                                    {renderItem(item, itemIndex)}
                                                </div>
                                            );
                                        })}
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                )}
            </div>
        </div>
    );
}
