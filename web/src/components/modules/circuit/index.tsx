import { useEffect, useState, useRef } from 'react';
import { useTranslations } from 'use-intl';
import {
    Zap,
    Hash,
    Timer,
    TimerOff,
    HelpCircle,
    RefreshCw,
    Power,
    PowerOff,
    TriangleAlert,
} from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { PageWrapper } from '@/components/common/PageWrapper';
import { useSettingList, useSetSetting, SettingKey } from '@/api/endpoints/setting';
import {
    useCircuitBreakerList,
    useCircuitBreakerManual,
    useCircuitBreakerReset,
    type CircuitBreaker,
} from '@/api/endpoints/circuitBreaker';
import { toast } from '@/components/common/Toast';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';

export function Circuit() {
    const t = useTranslations('circuit');

    return (
        <PageWrapper className="columns-1 gap-4 pb-24 *:mb-4 *:break-inside-avoid">
            <CircuitConfigCard />
            <CircuitBreakerTable />
        </PageWrapper>
    );
}

function CircuitConfigCard() {
    const t = useTranslations('circuit.config');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [threshold, setThreshold] = useState('');
    const [cooldown, setCooldown] = useState('');
    const [maxCooldown, setMaxCooldown] = useState('');

    const initialThreshold = useRef('');
    const initialCooldown = useRef('');
    const initialMaxCooldown = useRef('');

    useEffect(() => {
        if (settings) {
            const th = settings.find(s => s.key === SettingKey.CircuitBreakerThreshold);
            const cd = settings.find(s => s.key === SettingKey.CircuitBreakerCooldown);
            const mcd = settings.find(s => s.key === SettingKey.CircuitBreakerMaxCooldown);
            if (th) {
                queueMicrotask(() => setThreshold(th.value));
                initialThreshold.current = th.value;
            }
            if (cd) {
                queueMicrotask(() => setCooldown(cd.value));
                initialCooldown.current = cd.value;
            }
            if (mcd) {
                queueMicrotask(() => setMaxCooldown(mcd.value));
                initialMaxCooldown.current = mcd.value;
            }
        }
    }, [settings]);

    const handleSave = (key: string, value: string, initialValue: string) => {
        if (value === initialValue) return;
        setSetting.mutate({ key, value }, {
            onSuccess: () => {
                toast.success(t('saved'));
                if (key === SettingKey.CircuitBreakerThreshold) initialThreshold.current = value;
                else if (key === SettingKey.CircuitBreakerCooldown) initialCooldown.current = value;
                else if (key === SettingKey.CircuitBreakerMaxCooldown) initialMaxCooldown.current = value;
            }
        });
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Zap className="h-5 w-5" />
                {t('title')}
                <TooltipProvider>
                    <Tooltip>
                        <TooltipTrigger asChild>
                            <HelpCircle className="size-4 text-muted-foreground cursor-help" />
                        </TooltipTrigger>
                        <TooltipContent>{t('hint')}</TooltipContent>
                    </Tooltip>
                </TooltipProvider>
            </h2>

            <ConfigRow
                icon={<Hash className="h-5 w-5 text-muted-foreground" />}
                label={t('threshold.label')}
                placeholder={t('threshold.placeholder')}
                value={threshold}
                onChange={setThreshold}
                onBlur={() => handleSave(SettingKey.CircuitBreakerThreshold, threshold, initialThreshold.current)}
            />
            <ConfigRow
                icon={<Timer className="h-5 w-5 text-muted-foreground" />}
                label={t('cooldown.label')}
                placeholder={t('cooldown.placeholder')}
                value={cooldown}
                onChange={setCooldown}
                onBlur={() => handleSave(SettingKey.CircuitBreakerCooldown, cooldown, initialCooldown.current)}
            />
            <ConfigRow
                icon={<TimerOff className="h-5 w-5 text-muted-foreground" />}
                label={t('maxCooldown.label')}
                placeholder={t('maxCooldown.placeholder')}
                value={maxCooldown}
                onChange={setMaxCooldown}
                onBlur={() => handleSave(SettingKey.CircuitBreakerMaxCooldown, maxCooldown, initialMaxCooldown.current)}
            />
        </div>
    );
}

function ConfigRow({
    icon,
    label,
    placeholder,
    value,
    onChange,
    onBlur,
}: {
    icon: React.ReactNode;
    label: string;
    placeholder: string;
    value: string;
    onChange: (v: string) => void;
    onBlur: () => void;
}) {
    return (
        <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
                {icon}
                <span className="text-sm font-medium">{label}</span>
            </div>
            <Input type="number" value={value} onChange={(e) => onChange(e.target.value)} onBlur={onBlur} placeholder={placeholder} className="w-48 rounded-xl" />
        </div>
    );
}

function CircuitBreakerTable() {
    const t = useTranslations('circuit.manage');
    const { data: records = [] } = useCircuitBreakerList();
    const manual = useCircuitBreakerManual();
    const reset = useCircuitBreakerReset();

    const formatErrorTime = (sec: number) => {
        if (!sec) return '--';
        const d = new Date(sec * 1000);
        const pad = (n: number) => String(n).padStart(2, '0');
        return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
    };

    const handleManual = (record: CircuitBreaker, disabled: boolean) => {
        manual.mutate(
            { channel_id: record.channel_id, model_name: record.model_name, disabled },
            {
                onSuccess: () => toast.success(disabled ? t('disabledToast') : t('enabledToast')),
                onError: (e) => toast.error(e.message),
            }
        );
    };

    const handleReset = (record: CircuitBreaker) => {
        reset.mutate(
            { channel_id: record.channel_id, model_name: record.model_name },
            {
                onSuccess: () => toast.success(t('resetToast')),
                onError: (e) => toast.error(e.message),
            }
        );
    };

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-4">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <TriangleAlert className="h-5 w-5" />
                {t('title')}
            </h2>

            {records.length === 0 ? (
                <div className="rounded-2xl border border-border/70 bg-muted/30 p-4 text-sm text-muted-foreground">
                    {t('empty')}
                </div>
            ) : (
                <div className="rounded-2xl border border-border/70 overflow-x-auto">
                    <Table className="table-fixed">
                        <TableHeader>
                            <TableRow>
                                <TableHead className="w-[16%]">{t('channel')}</TableHead>
                                <TableHead className="w-[18%]">{t('model')}</TableHead>
                                <TableHead className="w-[18%]">{t('status')}</TableHead>
                                <TableHead className="w-[16%]">{t('lastErrorTime')}</TableHead>
                                <TableHead className="w-[18%]">{t('lastError')}</TableHead>
                                <TableHead className="text-right w-[14%]">{t('actions')}</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {records.map((r) => {
                                const tripping = !r.manual_disabled && r.state === 1;
                                const halfOpen = !r.manual_disabled && r.state === 2;
                                return (
                                    <TableRow key={`${r.channel_id}-${r.model_name}`}>
                                        <TableCell className="font-medium truncate">{r.channel_name}</TableCell>
                                        <TableCell className="truncate">{r.model_name}</TableCell>
                                        <TableCell>
                                            <div className="flex flex-wrap gap-1">
                                                {r.manual_disabled && (
                                                    <Badge variant="secondary" className="bg-destructive/15 text-destructive">{t('statusManualDisabled')}</Badge>
                                                )}
                                                {tripping && (
                                                    <Badge variant="secondary" className="bg-red-500/15 text-red-700 dark:text-red-400">{t('statusTripped')}</Badge>
                                                )}
                                                {halfOpen && (
                                                    <Badge variant="secondary" className="bg-orange-500/15 text-orange-700 dark:text-orange-400">{t('statusHalfOpen')}</Badge>
                                                )}
                                                {!r.manual_disabled && r.state === 0 && (
                                                    <Badge variant="secondary" className="bg-green-500/15 text-green-700 dark:text-green-400">{t('statusClosed')}</Badge>
                                                )}
                                                {r.trip_count > 0 && <Badge variant="secondary">x{r.trip_count}</Badge>}
                                            </div>
                                        </TableCell>
                                        <TableCell className="whitespace-nowrap text-xs text-muted-foreground truncate">
                                            {formatErrorTime(r.last_error_time)}
                                        </TableCell>
                                        <TableCell className="p-0">
                                            {r.last_error ? (
                                                <TooltipProvider>
                                                    <Tooltip>
                                                        <TooltipTrigger asChild>
                                                            <span className="block w-full truncate px-4 py-4 text-xs text-muted-foreground cursor-help">{r.last_error}</span>
                                                        </TooltipTrigger>
                                                        <TooltipContent className="max-w-md break-words">{r.last_error}</TooltipContent>
                                                    </Tooltip>
                                                </TooltipProvider>
                                            ) : (
                                                <span className="block px-4 py-4 text-xs text-muted-foreground">--</span>
                                            )}
                                        </TableCell>
                                        <TableCell>
                                            <div className="flex items-center justify-end gap-1">
                                                {r.manual_disabled ? (
                                                    <Button size="sm" variant="outline" className="h-7 px-2 text-xs" onClick={() => handleManual(r, false)} disabled={manual.isPending}>
                                                        <Power className="size-3.5 mr-1" />
                                                        {t('enable')}
                                                    </Button>
                                                ) : (
                                                    <Button size="sm" variant="outline" className="h-7 px-2 text-xs" onClick={() => handleManual(r, true)} disabled={manual.isPending}>
                                                        <PowerOff className="size-3.5 mr-1" />
                                                        {t('disable')}
                                                    </Button>
                                                )}
                                                {(tripping || halfOpen) && (
                                                    <Button size="sm" variant="outline" className="h-7 px-2 text-xs" onClick={() => handleReset(r)} disabled={reset.isPending}>
                                                        <RefreshCw className="size-3.5 mr-1" />
                                                        {t('reset')}
                                                    </Button>
                                                )}
                                            </div>
                                        </TableCell>
                                    </TableRow>
                                );
                            })}
                        </TableBody>
                    </Table>
                </div>
            )}
        </div>
    );
}