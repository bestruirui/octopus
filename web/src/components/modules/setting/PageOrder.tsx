'use client';

import { useEffect, useRef, useState } from 'react';
import {
    DragDropContext,
    Draggable,
    Droppable,
    type DropResult,
} from '@hello-pangea/dnd';
import { GripVertical, ListOrdered, RotateCcw } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { SettingKey, useSetSetting, useSettingList } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';
import { Button } from '@/components/ui/button';
import {
    DEFAULT_NAV_ORDER,
    parseNavOrder,
    serializeNavOrder,
} from '@/components/modules/navbar/nav-order';

function reorderNavOrder(list: string[], startIndex: number, endIndex: number): string[] {
    const next = [...list];
    const [removed] = next.splice(startIndex, 1);
    next.splice(endIndex, 0, removed);
    return next;
}

export function SettingPageOrder() {
    const t = useTranslations('setting');
    const tNavbar = useTranslations('navbar');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const defaultSerialized = serializeNavOrder(DEFAULT_NAV_ORDER);
    const [navOrder, setNavOrder] = useState<string[]>([...DEFAULT_NAV_ORDER]);
    const [intendedValue, setIntendedValue] = useState(defaultSerialized);
    const intendedValueRef = useRef(defaultSerialized);
    const hasLoadedRef = useRef(false);
    const hasLocalIntentRef = useRef(false);
    const isSavingRef = useRef(false);

    useEffect(() => {
        if (!settings) return;

        const navOrderSetting = settings.find((item) => item.key === SettingKey.NavOrder);
        const serverValue = navOrderSetting?.value || defaultSerialized;
        const nextOrder = parseNavOrder(serverValue);

        queueMicrotask(() => {
            if (hasLoadedRef.current && hasLocalIntentRef.current && serverValue !== intendedValueRef.current) {
                return;
            }

            hasLoadedRef.current = true;
            if (serverValue === intendedValueRef.current) {
                hasLocalIntentRef.current = false;
            }
            intendedValueRef.current = serverValue;
            setIntendedValue(serverValue);
            setNavOrder(nextOrder);
        });
    }, [defaultSerialized, settings]);

    const flushNavOrderSave = () => {
        if (isSavingRef.current || !hasLocalIntentRef.current) {
            return;
        }

        const value = intendedValueRef.current;
        isSavingRef.current = true;

        setSetting.mutate(
            { key: SettingKey.NavOrder, value },
            {
                onSuccess: () => {
                    toast.success(t('saved'));
                },
                onError: () => {
                    if (intendedValueRef.current === value) {
                        hasLocalIntentRef.current = false;
                    }
                },
                onSettled: () => {
                    isSavingRef.current = false;
                    if (hasLocalIntentRef.current && intendedValueRef.current !== value) {
                        flushNavOrderSave();
                    }
                },
            },
        );
    };

    const saveNavOrder = (nextOrder: string[]) => {
        const normalized = parseNavOrder(JSON.stringify(nextOrder));
        const serialized = serializeNavOrder(normalized);

        setNavOrder(normalized);
        if (serialized === intendedValueRef.current) return;

        intendedValueRef.current = serialized;
        hasLocalIntentRef.current = true;
        setIntendedValue(serialized);
        flushNavOrderSave();
    };

    const handleDragEnd = (result: DropResult) => {
        const { destination, source } = result;
        if (!destination || destination.index === source.index) {
            return;
        }

        saveNavOrder(reorderNavOrder(navOrder, source.index, destination.index));
    };

    const isResetDisabled =
        serializeNavOrder(navOrder) === defaultSerialized &&
        intendedValue === defaultSerialized;

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <div className="flex items-start justify-between gap-4">
                <div className="space-y-1">
                    <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                        <ListOrdered className="h-5 w-5" />
                        {t('pageOrder.title')}
                    </h2>
                    <p className="text-sm text-muted-foreground">
                        {t('pageOrder.description')}
                    </p>
                </div>
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => saveNavOrder([...DEFAULT_NAV_ORDER])}
                    disabled={isResetDisabled}
                    className="rounded-xl"
                >
                    <RotateCcw className="h-4 w-4" />
                    {t('pageOrder.reset')}
                </Button>
            </div>

            <div className="rounded-2xl border border-border/60 bg-background/70">
                <DragDropContext onDragEnd={handleDragEnd}>
                    <Droppable droppableId="page-order">
                        {(droppableProvided) => (
                            <div
                                ref={droppableProvided.innerRef}
                                {...droppableProvided.droppableProps}
                                className="p-2 space-y-2"
                            >
                                {navOrder.map((id, index) => (
                                    <Draggable key={id} draggableId={id} index={index}>
                                        {(draggableProvided, snapshot) => (
                                            <div
                                                ref={draggableProvided.innerRef}
                                                {...draggableProvided.draggableProps}
                                                className="flex items-center gap-3 rounded-2xl border border-border/60 bg-card px-3 py-3"
                                                style={{
                                                    ...draggableProvided.draggableProps.style,
                                                    ...(snapshot.isDragging
                                                        ? { boxShadow: '0 8px 24px rgba(0,0,0,0.12)' }
                                                        : null),
                                                }}
                                            >
                                                <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                                                    {index + 1}
                                                </span>
                                                <div
                                                    className="rounded-md p-1 text-muted-foreground hover:bg-muted"
                                                    {...draggableProvided.dragHandleProps}
                                                >
                                                    <GripVertical className="h-4 w-4" />
                                                </div>
                                                <span className="text-sm font-medium text-card-foreground">
                                                    {tNavbar(id)}
                                                </span>
                                            </div>
                                        )}
                                    </Draggable>
                                ))}
                                {droppableProvided.placeholder}
                            </div>
                        )}
                    </Droppable>
                </DragDropContext>
            </div>

            <p className="text-xs text-muted-foreground">
                {t('pageOrder.hint')}
            </p>
        </div>
    );
}
