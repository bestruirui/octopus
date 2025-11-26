'use client';

import { useStatsDaily, type StatsDaily } from '@/api/endpoints/stats';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';
import { useMemo, useRef, useLayoutEffect, useState, useCallback } from 'react';
import { useTranslations } from 'next-intl';
import { Fragment } from 'react';
interface DayData {
    dateStr: string;
    isFuture: boolean;
    data: StatsDaily | null;
}

const ACTIVITY_LEVELS = [
    { min: 5000, level: 4 },
    { min: 2000, level: 3 },
    { min: 1000, level: 2 },
    { min: 1, level: 1 }
];

function getActivityLevel(value: number): number {
    if (value === 0) return 0;
    return ACTIVITY_LEVELS.find(level => value >= level.min)?.level || 1;
}

export function Activity() {
    const { data: statsDaily, isLoading } = useStatsDaily();
    const scrollRef = useRef<HTMLDivElement>(null);
    const t = useTranslations('activity');

    const [tooltip, setTooltip] = useState<{ day: DayData; x: number; y: number; visible: boolean } | null>(null);

    const days = useMemo(() => {
        if (!statsDaily) return [];

        const statsMap = new Map(statsDaily.map(stat => [stat.date.split('T')[0], stat]));

        const today = new Date();
        const todayStr = today.toISOString().slice(0, 10);

        today.setDate(today.getDate() - (today.getDay() + 53 * 7));

        const result: DayData[] = [];

        for (let i = 0; i < 54 * 7; i++) {
            const dateStr = today.toISOString().slice(0, 10);
            result.push({
                dateStr,
                isFuture: dateStr > todayStr,
                data: statsMap.get(dateStr) || null
            });
            today.setDate(today.getDate() + 1);
        }

        return result;
    }, [statsDaily]);

    const [maskImage, setMaskImage] = useState('none');

    const checkScroll = useCallback(() => {
        if (!scrollRef.current) return;
        const { scrollLeft, scrollWidth, clientWidth } = scrollRef.current;
        const isStart = scrollLeft <= 1;
        const isEnd = Math.abs(scrollWidth - clientWidth - scrollLeft) <= 1;

        if (isStart && isEnd) {
            setMaskImage('none');
        } else if (isStart) {
            setMaskImage('linear-gradient(to left, transparent, rgba(0,0,0,0) 10px, black 40px)');
        } else if (isEnd) {
            setMaskImage('linear-gradient(to right, transparent, rgba(0,0,0,0) 10px,black 40px)');
        } else {
            setMaskImage('linear-gradient(to right, transparent, rgba(0,0,0,0) 10px, black 40px, black calc(100% - 40px),  rgba(0,0,0,0) calc(100% - 10px), transparent)');
        }
    }, []);

    useLayoutEffect(() => {
        const scrollToRight = () => {
            if (scrollRef.current) {
                scrollRef.current.scrollLeft = scrollRef.current.scrollWidth;
                checkScroll();
            }
        };
        scrollToRight();
        window.addEventListener('resize', scrollToRight);
        return () => window.removeEventListener('resize', scrollToRight);
    }, [days, isLoading, checkScroll]);

    return (
        <div className="rounded-3xl bg-card border-card-border border custom-shadow">
            <div
                ref={scrollRef}
                onScroll={checkScroll}
                className="overflow-x-auto p-4"
                style={{ maskImage, WebkitMaskImage: maskImage }}
            >
                <div className="ml-auto w-fit">
                    <div className="grid gap-1"
                        style={{
                            gridTemplateColumns: 'repeat(54, 0.875rem)',
                            gridTemplateRows: 'repeat(7, 0.875rem)',
                            gridAutoFlow: 'column'
                        }}
                    >
                        {days.map((day) => {
                            if (day.isFuture) {
                                return <div key={day.dateStr} />;
                            }

                            const requestCount = day.data?.request_count || 0;
                            const level = getActivityLevel(requestCount);

                            return (
                                <div
                                    key={day.dateStr}
                                    className="rounded-sm transition-all cursor-pointer hover:scale-150"
                                    onMouseEnter={(e) => {
                                        const rect = e.currentTarget.getBoundingClientRect();
                                        setTooltip({ day, x: rect.left + rect.width / 2, y: rect.top, visible: true });
                                    }}
                                    onMouseLeave={() => setTooltip(prev => prev ? { ...prev, visible: false } : null)}
                                    style={{ backgroundColor: level === 0 ? 'var(--muted)' : `color-mix(in oklch, var(--primary) ${level * 25}%, var(--muted))` }}
                                />
                            );
                        })}
                    </div>
                </div>
            </div>
            {tooltip && (() => {
                const isLeft = tooltip.x < 200;
                const isRight = tooltip.x > window.innerWidth - 200;
                const isTop = tooltip.y < window.innerHeight / 2;

                let transform = 'translate(-50%, 15%)';
                if (!isTop && !isLeft && !isRight) {
                    transform = 'translate(-50%, -105%)';
                } else if (isTop && isLeft) {
                    transform = 'translate(10%, 15%)';
                } else if (isTop && isRight) {
                    transform = 'translate(-110%, 15%)';
                } else if (!isTop && isLeft) {
                    transform = 'translate(10%, -105%)';
                } else if (!isTop && isRight) {
                    transform = 'translate(-110%, -105%)';
                }

                return (
                    <div
                        className={`fixed z-50 w-fit min-w-max text-sm bg-popover text-popover-foreground border rounded-3xl custom-shadow p-3 transition-opacity duration-500 pointer-events-none ${tooltip.visible ? 'opacity-100' : 'opacity-0'}`}
                        style={{
                            left: tooltip.x,
                            top: tooltip.y,
                            transform
                        }}
                    >
                        <div className="space-y-2">
                            <p className="font-semibold text-foreground">{tooltip.day.dateStr}</p>
                            {tooltip.day.data ? (
                                <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 items-center text-muted-foreground">
                                    {[
                                        { labelKey: 'requestCount', ...formatCount(tooltip.day.data.request_count) },
                                        { labelKey: 'waitTime', ...formatTime(tooltip.day.data.wait_time) },
                                        { labelKey: 'inputToken', ...formatCount(tooltip.day.data.input_token) },
                                        { labelKey: 'inputCost', ...formatMoney(tooltip.day.data.input_cost) },
                                        { labelKey: 'outputToken', ...formatCount(tooltip.day.data.output_token) },
                                        { labelKey: 'outputCost', ...formatMoney(tooltip.day.data.output_cost) },
                                    ].map((item, index) => (
                                        <Fragment key={index}>
                                            <span className="wrap-break-word">{t(item.labelKey)}</span>
                                            <span className="text-foreground font-medium text-right">{item.value}{item.unit}</span>
                                        </Fragment>
                                    ))}
                                </div>
                            ) : (
                                <p className="text-muted-foreground">{t('noData')}</p>
                            )}
                        </div>
                    </div>
                );
            })()}
        </div>
    );
}
