import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'use-intl';
import { FlaskConical, HelpCircle } from 'lucide-react';
import { useSettingList, useSetSetting, SettingKey } from '@/api/setting';
import { toast } from 'sonner';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';

// 未在服务端配置测试提示词时展示的默认文案，对所有语言一致。
const DEFAULT_TEST_PROMPT = '请简洁回答：17 × 23 等于多少？给出计算过程。';

// SettingTestPrompt 提供测试提示词的编辑与保存。
export function SettingTestPrompt() {
    const t = useTranslations('setting');
    const { data: settings } = useSettingList();
    const setSetting = useSetSetting();

    const [testPrompt, setTestPrompt] = useState('');
    const [loaded, setLoaded] = useState(false);
    const initialTestPrompt = useRef('');

    useEffect(() => {
        // 仅在首次取得服务端设置时初始化，避免周期性刷新覆盖正在编辑的内容。
        if (!settings || loaded) return;
        const setting = settings.find(s => s.key === SettingKey.TestPrompt);
        const value = setting ? setting.value : DEFAULT_TEST_PROMPT;
        queueMicrotask(() => {
            setTestPrompt(value);
            setLoaded(true);
        });
        initialTestPrompt.current = value;
    }, [settings, loaded]);

    const handleSave = () => {
        // 设置未加载完成或上一次保存仍在进行时不允许保存，避免用默认值覆盖服务端或并发乱序。
        if (!loaded || setSetting.isPending) return;
        if (testPrompt === initialTestPrompt.current) return;

        setSetting.mutate({ key: SettingKey.TestPrompt, value: testPrompt }, {
            onSuccess: () => {
                toast.success(t('saved'));
                initialTestPrompt.current = testPrompt;
            },
            onError: () => {
                // 保存失败不更新基线，失焦后再次编辑即可重试。
                toast.error(t('testPrompt.saveFailed'));
            }
        });
    };

    return (
        <div className="space-y-5 rounded-3xl border border-border bg-card p-6">
            <h2 className="flex items-center gap-2 text-lg font-bold text-card-foreground">
                <FlaskConical className="size-5" />
                {t('testPrompt.title')}
                <Tooltip>
                    <TooltipTrigger asChild>
                        <HelpCircle className="size-4 text-muted-foreground cursor-help" />
                    </TooltipTrigger>
                    <TooltipContent side="top" sideOffset={10} align="center">
                        {t('testPrompt.hint')}
                    </TooltipContent>
                </Tooltip>
            </h2>
            <textarea
                value={testPrompt}
                onChange={(e) => setTestPrompt(e.target.value)}
                onBlur={handleSave}
                disabled={!loaded || setSetting.isPending}
                aria-label={t('testPrompt.title')}
                placeholder={t('testPrompt.placeholder')}
                className="min-h-24 w-full rounded-xl border border-border bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
            />
        </div>
    );
}
