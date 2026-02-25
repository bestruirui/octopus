import { useState, useRef, useEffect } from 'react';
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

import { FileUp, ChevronLeft, ChevronRight } from 'lucide-react';
import { useTranslations } from 'next-intl';

interface BatchImportModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    onKeysImported?: (keys: string[]) => void;
}

/**
 * PaginatedList 组件 - 分页列表辅助组件
 * 用于预览导入的密钥列表
 */
const PaginatedList = ({ items }: { items: string[] }) => {
    const t = useTranslations('channel.batchImport');
    const [page, setPage] = useState(1);
    const [inputPage, setInputPage] = useState('1');
    const pageSize = 10;
    const totalPages = Math.max(1, Math.ceil(items.length / pageSize));
    
    // 当项目变化时重置页码
    useEffect(() => {
        setPage(1);
        setInputPage('1');
    }, [items]);

    const start = (page - 1) * pageSize;
    const end = start + pageSize;
    const currentItems = items.slice(start, end);

    const handlePageInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setInputPage(e.target.value);
    };

    const handlePageInputBlur = () => {
        const pageNum = parseInt(inputPage, 10);
        if (!isNaN(pageNum) && pageNum >= 1 && pageNum <= totalPages) {
            setPage(pageNum);
        } else {
            setInputPage(page.toString());
        }
    };

    const handlePageInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
            handlePageInputBlur();
        }
    };

    const handlePageChange = (newPage: number) => {
        setPage(newPage);
        setInputPage(newPage.toString());
    };

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
                            <div key={start + i} className="flex items-center text-sm font-mono border-b last:border-0 py-1.5 px-2 hover:bg-muted/50 transition-colors min-w-0">
                                <span className="w-8 text-xs text-muted-foreground select-none shrink-0 text-right mr-3">
                                    {start + i + 1}.
                                </span>
                                <span className="truncate flex-1 text-foreground/90 min-w-0" title={k}>{k}</span>
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
                            onClick={() => handlePageChange(Math.max(1, page - 1))}
                            disabled={page === 1}
                        >
                            <ChevronLeft className="h-4 w-4" />
                        </Button>
                        <div className="flex items-center gap-1">
                            <input
                                type="text"
                                value={inputPage}
                                onChange={handlePageInputChange}
                                onBlur={handlePageInputBlur}
                                onKeyDown={handlePageInputKeyDown}
                                className="w-10 h-7 text-center text-xs border rounded px-1 bg-background"
                            />
                            <span className="text-muted-foreground">/ {totalPages}</span>
                        </div>
                        <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7"
                            onClick={() => handlePageChange(Math.min(totalPages, page + 1))}
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

/**
 * BatchImportModal 组件 - 批量导入密钥对话框
 * 
 * 功能特性：
 * - 支持文本输入和文件上传
 * - 实时预览解析的密钥
 */
export function BatchImportModal({ open, onOpenChange, onKeysImported }: BatchImportModalProps) {
    const t = useTranslations('channel.batchImport');
    const [input, setInput] = useState('');
    const [parsedKeys, setParsedKeys] = useState<string[]>([]);
    const fileInputRef = useRef<HTMLInputElement>(null);

    // 打开时重置状态
    useEffect(() => {
        if (open) {
            setInput('');
            setParsedKeys([]);
        }
    }, [open]);

    // 输入变化时解析密钥
    useEffect(() => {
        const keys = input
            .split(/[\n,]+/)
            .map(k => k.trim())
            .filter(k => k.length > 0);
        // 去重以便预览
        setParsedKeys(Array.from(new Set(keys)));
    }, [input]);

    // 处理文件上传
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
        // 重置值以便可以再次选择相同文件
        e.target.value = '';
    };

    // 处理导入
    const handleImport = () => {
        if (parsedKeys.length === 0) return;
        
        // 统一使用本地模式：将密钥添加到表单中，由用户点击保存时才真正保存到数据库
        if (onKeysImported) {
            onKeysImported(parsedKeys);
            onOpenChange(false);
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[600px] max-w-[95vw] overflow-hidden">
                <DialogHeader>
                    <DialogTitle>{t('title')}</DialogTitle>
                    <DialogDescription>
                        {t('description')}
                    </DialogDescription>
                </DialogHeader>

                <div className="space-y-4 min-w-0 overflow-hidden">
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
                        className="h-[150px] font-mono text-xs resize-none overflow-auto break-all [field-sizing:initial] w-full max-w-full"
                    />

                    {parsedKeys.length > 0 && (
                        <div className="space-y-2">
                            <h4 className="text-sm font-medium">{t('preview')}</h4>
                            <PaginatedList items={parsedKeys} />
                        </div>
                    )}
                </div>

                <DialogFooter>
                    <Button variant="ghost" onClick={() => onOpenChange(false)}>
                        {t('cancel')}
                    </Button>
                    <Button onClick={handleImport} disabled={parsedKeys.length === 0}>
                        {t('addToForm')}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
