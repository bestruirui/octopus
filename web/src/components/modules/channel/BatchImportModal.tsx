import { useState, useRef, useEffect, useMemo } from 'react';
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
    DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';

import { Progress } from '@/components/ui/progress';
import { useBatchImportKeys, useGetBatchImportStatus, useCancelBatchImport, BatchImportStatusResponse } from '@/api/endpoints/channel';
import { toast } from 'sonner';
import { Download, AlertCircle, CheckCircle2, Loader2, FileUp, RefreshCw, XCircle, ChevronLeft, ChevronRight } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { ScrollArea } from '@/components/ui/scroll-area';

interface BatchImportModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    channelId?: number;
    onSuccess?: () => void;
    onKeysImported?: (keys: string[]) => void;
}

// Pagination list helper for preview
const PaginatedList = ({ items }: { items: string[] }) => {
    const t = useTranslations('channel.batchImport');
    const [page, setPage] = useState(1);
    const pageSize = 10;
    const totalPages = Math.max(1, Math.ceil(items.length / pageSize));
    
    // Reset page if items change
    useEffect(() => {
        setPage(1);
    }, [items]);

    const start = (page - 1) * pageSize;
    const end = start + pageSize;
    const currentItems = items.slice(start, end);

    return (
        <div className="space-y-2">
            <div className="border rounded-md h-[200px] overflow-y-auto bg-muted/10 p-2">
                {items.length === 0 ? (
                    <div className="flex h-full items-center justify-center text-muted-foreground text-sm">
                        {t('noKeys')}
                    </div>
                ) : (
                    <div className="space-y-1">
                        {currentItems.map((k, i) => (
                            <div key={start + i} className="flex items-center text-sm font-mono border-b last:border-0 py-1.5 px-2 hover:bg-muted/50 transition-colors">
                                <span className="w-8 text-xs text-muted-foreground select-none shrink-0 text-right mr-3">
                                    {start + i + 1}.
                                </span>
                                <span className="truncate flex-1 text-foreground/90" title={k}>{k}</span>
                            </div>
                        ))}
                    </div>
                )}
            </div>
            
            {items.length > 0 && (
                <div className="flex items-center justify-between text-xs px-1">
                    <div className="text-muted-foreground">
                        {t('keyCount', { total: items.length })}
                    </div>
                    <div className="flex items-center gap-2">
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7"
                            onClick={() => setPage(p => Math.max(1, p - 1))}
                            disabled={page === 1}
                        >
                            <ChevronLeft className="h-4 w-4" />
                        </Button>
                        <span className="min-w-[3rem] text-center font-medium">
                            {page} / {totalPages}
                        </span>
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7"
                            onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                            disabled={page === totalPages}
                        >
                            <ChevronRight className="h-4 w-4" />
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
};

export function BatchImportModal({ open, onOpenChange, channelId, onSuccess, onKeysImported }: BatchImportModalProps) {
    const t = useTranslations('channel.batchImport');
    const [input, setInput] = useState('');
    const [parsedKeys, setParsedKeys] = useState<string[]>([]);
    const [step, setStep] = useState<'input' | 'processing' | 'result'>('input');
    const [jobId, setJobId] = useState<string | null>(null);
    const [progress, setProgress] = useState<BatchImportStatusResponse | null>(null);
    const fileInputRef = useRef<HTMLInputElement>(null);

    const batchImportMutation = useBatchImportKeys();
    const statusMutation = useGetBatchImportStatus();
    const cancelMutation = useCancelBatchImport();

    // Reset state when opening
    useEffect(() => {
        if (open) {
            setInput('');
            setParsedKeys([]);
            setStep('input');
            setJobId(null);
            setProgress(null);
        }
    }, [open]);

    // Parse input on change
    useEffect(() => {
        const keys = input
            .split(/[\n,]+/)
            .map(k => k.trim())
            .filter(k => k.length > 0);
        // Deduplicate locally for preview if needed, but backend handles it too.
        // Let's keep all for now or unique them? 
        // User might want to see count.
        setParsedKeys(Array.from(new Set(keys)));
    }, [input]);

    const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;

        const reader = new FileReader();
        reader.onload = (event) => {
            const text = event.target?.result as string;
            if (text) {
                setInput(prev => prev + (prev ? '\n' : '') + text);
            }
        };
        reader.readAsText(file);
        // Reset value so same file can be selected again
        e.target.value = '';
    };

    const handleImport = () => {
        if (parsedKeys.length === 0) return;
        
        if (!channelId) {
            // Local mode: just pass keys back
            if (onKeysImported) {
                onKeysImported(parsedKeys);
                onOpenChange(false);
            }
            return;
        }

        batchImportMutation.mutate(
            { channel_id: channelId, keys: parsedKeys },
            {
                onSuccess: (data) => {
                    setJobId(data.job_id);
                    setStep('processing');
                },
                onError: (error) => {
                    toast.error(t('startFailed'));
                }
            }
        );
    };

    // Polling status
    useEffect(() => {
        if (step !== 'processing' || !jobId) return;

        const interval = setInterval(() => {
            statusMutation.mutate(jobId, {
                onSuccess: (data) => {
                    setProgress(data);
                    if (data.status === 'completed' || data.status === 'failed') {
                        setStep('result');
                        clearInterval(interval);
                        if (data.status === 'completed') {
                            onSuccess?.();
                        }
                    }
                },
                onError: () => {
                    // Don't stop polling immediately on one error, but maybe warn?
                }
            });
        }, 1000);

        return () => clearInterval(interval);
    }, [step, jobId, statusMutation, onSuccess]);

    const downloadReport = () => {
        if (!progress) return;
        
        let content = `Import Report for Channel ${channelId}\n`;
        content += `Date: ${new Date().toLocaleString()}\n`;
        content += `Total: ${progress.total}, Success: ${progress.success_count}, Failed: ${progress.fail_count}\n\n`;
        
        if (progress.duplicates && progress.duplicates.length > 0) {
            content += `Duplicates (${progress.duplicates.length}):\n`;
            content += progress.duplicates.join('\n') + '\n\n';
        }

        if (progress.errors && progress.errors.length > 0) {
            content += `Errors (${progress.errors.length}):\n`;
            content += progress.errors.join('\n') + '\n\n';
        }

        const blob = new Blob([content], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `import_report_${jobId}.txt`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    };

    const handleCancel = () => {
        if (!jobId) return;
        cancelMutation.mutate(jobId, {
            onSuccess: () => {
                toast.info(t('cancelling'));
            },
            onError: () => {
                toast.error(t('startFailed'));
            }
        });
    };

    const handleRetry = () => {
        setStep('input');
        setJobId(null);
        setProgress(null);
    };

    return (
        <Dialog open={open} onOpenChange={(val) => {
            if (step === 'processing') {
                // Prevent closing while processing? Or confirm cancellation?
                // For now, just allow closing but job continues in background
            }
            onOpenChange(val);
        }}>
            <DialogContent className="sm:max-w-[600px]">
                <DialogHeader>
                    <DialogTitle>{t('title')}</DialogTitle>
                    <DialogDescription>
                        {step === 'input' && t('description')}
                        {step === 'processing' && t('processingDescription')}
                        {step === 'result' && t('resultDescription')}
                    </DialogDescription>
                </DialogHeader>

                {step === 'input' && (
                    <div className="space-y-4">
                        <div className="flex justify-between items-center">
                            <span className="text-sm text-muted-foreground">
                                {t('keyCount', { total: parsedKeys.length })}
                            </span>
                            <div>
                                <input
                                    type="file"
                                    ref={fileInputRef}
                                    className="hidden"
                                    accept=".txt,.csv"
                                    onChange={handleFileUpload}
                                />
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() => fileInputRef.current?.click()}
                                >
                                    <FileUp className="w-4 h-4 mr-2" />
                                    {t('uploadFile')}
                                </Button>
                            </div>
                        </div>

                        <Textarea
                            value={input}
                            onChange={(e) => setInput(e.target.value)}
                            placeholder={t('placeholder')}
                            className="h-[150px] font-mono text-xs"
                        />

                        {parsedKeys.length > 0 && (
                            <div className="space-y-2">
                                <h4 className="text-sm font-medium">{t('preview')}</h4>
                                <PaginatedList items={parsedKeys} />
                            </div>
                        )}
                    </div>
                )}

                {step === 'processing' && (
                    <div className="space-y-6 py-8">
                        <div className="space-y-2">
                            <div className="flex justify-between text-sm">
                                <span>{t('progress')}</span>
                                <span>{progress ? Math.round((progress.processed / progress.total) * 100) : 0}%</span>
                            </div>
                            <Progress value={progress ? (progress.processed / progress.total) * 100 : 0} />
                        </div>
                        <div className="text-center text-sm text-muted-foreground">
                            {progress ? t('processingStats', { 
                                processed: progress.processed, 
                                total: progress.total 
                            }) : t('initializing')}
                        </div>
                    </div>
                )}

                {step === 'result' && progress && (
                    <div className="space-y-4">
                        <div className="grid grid-cols-3 gap-4 text-center">
                            <div className="p-4 bg-green-50 rounded-lg dark:bg-green-900/20">
                                <div className="text-2xl font-bold text-green-600 dark:text-green-400">
                                    {progress.success_count}
                                </div>
                                <div className="text-xs text-muted-foreground">{t('success')}</div>
                            </div>
                            <div className="p-4 bg-red-50 rounded-lg dark:bg-red-900/20">
                                <div className="text-2xl font-bold text-red-600 dark:text-red-400">
                                    {progress.fail_count}
                                </div>
                                <div className="text-xs text-muted-foreground">{t('failed')}</div>
                            </div>
                            <div className="p-4 bg-blue-50 rounded-lg dark:bg-blue-900/20">
                                <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">
                                    {progress.total}
                                </div>
                                <div className="text-xs text-muted-foreground">{t('total')}</div>
                            </div>
                        </div>

                        {(progress.duplicates.length > 0 || progress.errors.length > 0) && (
                            <ScrollArea className="h-[150px] w-full rounded-md border p-4 text-xs font-mono bg-muted/50">
                                {progress.duplicates.length > 0 && (
                                    <div className="mb-4">
                                        <div className="font-semibold text-yellow-600 dark:text-yellow-400 mb-1">Duplicates ({progress.duplicates.length}):</div>
                                        {progress.duplicates.map((k, i) => (
                                            <div key={i} className="text-muted-foreground truncate">{k}</div>
                                        ))}
                                    </div>
                                )}
                                {progress.errors.length > 0 && (
                                    <div>
                                        <div className="font-semibold text-red-600 dark:text-red-400 mb-1">Errors ({progress.errors.length}):</div>
                                        {progress.errors.map((e, i) => (
                                            <div key={i} className="text-red-500 truncate">{e}</div>
                                        ))}
                                    </div>
                                )}
                            </ScrollArea>
                        )}

                        {(progress.fail_count > 0 || progress.duplicates.length > 0) && (
                            <Button 
                                variant="outline" 
                                className="w-full"
                                onClick={downloadReport}
                            >
                                <Download className="w-4 h-4 mr-2" />
                                {t('downloadReport')}
                            </Button>
                        )}
                    </div>
                )}

                <DialogFooter>
                    {step === 'input' && (
                        <>
                            <Button variant="ghost" onClick={() => onOpenChange(false)}>
                                {t('cancel')}
                            </Button>
                            <Button onClick={handleImport} disabled={parsedKeys.length === 0 || batchImportMutation.isPending}>
                                {batchImportMutation.isPending && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                                {channelId ? t('import') : t('addToForm')}
                            </Button>
                        </>
                    )}
                    {step === 'processing' && (
                        <>
                            <Button variant="ghost" onClick={handleCancel} disabled={cancelMutation.isPending}>
                                <XCircle className="w-4 h-4 mr-2" />
                                {t('cancel')}
                            </Button>
                            <Button variant="secondary" disabled>
                                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                                {t('processing')}
                            </Button>
                        </>
                    )}
                    {step === 'result' && (
                        <>
                            <Button variant="ghost" onClick={handleRetry}>
                                <RefreshCw className="w-4 h-4 mr-2" />
                                {t('retry')}
                            </Button>
                            <Button onClick={() => onOpenChange(false)}>
                                {t('close')}
                            </Button>
                        </>
                    )}
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
