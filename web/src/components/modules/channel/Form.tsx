import { useState } from 'react';
import { ChevronLeft, Plus, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { useTranslations } from 'use-intl';
import {
    type ChannelDetail,
    useChannelDetail,
    useCreateChannel,
    useUpdateChannel,
} from '@/api/channel';
import { CHANNEL_PRESETS, type ChannelPreset } from '@/lib/channel-presets';
import { useMorphingDialog } from '@/components/ui/morphing-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { FormGrants } from './FormGrants';
import { FormKeys } from './FormKeys';
import { FormOverrides } from './FormOverrides';
import { FormHeaders } from './FormHeaders';
import { IconButton } from '@/components/common/IconButton';
import { useSettingStore } from '@/stores/setting';
import {
    emptyFormState,
    fromChannel,
    toChannelDetail,
    type ChannelFormState,
} from './state';

type StepID = 'preset' | 'connection' | 'keys' | 'grants' | 'advanced';

// ChannelForm 新建与编辑共用; 编辑时按 id 取回整份配置, 到齐后再进表单。
// 配置不随渠道列表下发, 故编辑必然要等这一趟请求; 占位与表单同高, 弹窗不会因此跳动。
export function ChannelForm({ channelId, onBack }: {
    channelId?: number;
    onBack?: () => void; // 编辑态由统计视图切入, 给出返回入口; 新建态没有可返回的视图。
}) {
    const t = useTranslations('channel.form');
    const { data: detail, isPending, isError } = useChannelDetail(channelId);

    if (channelId === undefined) {
        return <ChannelFormFields />;
    }
    if (!detail) {
        return (
            <div className="flex items-center justify-center h-[min(29rem,calc(100vh-10rem))]">
                <p className="text-sm text-muted-foreground">
                    {isError ? t('detailFailed') : isPending ? t('detailLoading') : null}
                </p>
            </div>
        );
    }
    return <ChannelFormFields channel={detail} onBack={onBack} />;
}

// ChannelFormFields 承载表单本体; 初始状态在挂载时定稿, 故必须等配置到齐后才渲染。
// 新建从模板步骤起, 编辑跳过模板直接进连接步骤; 步骤顺序即数据依赖顺序:
// 模板给出地址与路径 -> 拉取需要地址与凭据 -> 授权引用凭据与模型。
function ChannelFormFields({ channel, onBack }: { channel?: ChannelDetail; onBack?: () => void }) {
    const t = useTranslations('channel.form');
    const locale = useSettingStore((state) => state.locale);
    const { setIsOpen } = useMorphingDialog();
    const createChannel = useCreateChannel();
    const updateChannel = useUpdateChannel();
    const [state, setState] = useState<ChannelFormState>(channel ? fromChannel(channel) : emptyFormState);
    const [step, setStep] = useState<StepID>(channel ? 'connection' : 'preset');
    const isPending = createChannel.isPending || updateChannel.isPending;
    // 新建与编辑弹窗可同时存在, 控件 id 需按渠道隔离。
    const idPrefix = channel ? `channel-${channel.id}` : 'new-channel';

    // 后端会拒的四项在此先挡: 名称与地址非空, 路径以 / 开头, 至少一份填了 Key 的凭据, 至少一个模型。
    const canSubmit = state.name.trim() !== ''
        && state.base_url.trim() !== ''
        && [state.openai_chat_completion_path, state.openai_response_path, state.anthropic_message_path]
            .every((path) => path === '' || path.startsWith('/'))
        && state.keys.length > 0
        && state.keys.every((k) => k.key.trim() !== '')
        && state.models.length > 0;

    const steps: { id: StepID; label: string }[] = [
        ...(channel ? [] : [{ id: 'preset' as StepID, label: t('stepPreset') }]),
        { id: 'connection', label: t('stepConnection') },
        { id: 'keys', label: t('stepKeys') },
        { id: 'grants', label: t('stepGrants') },
        { id: 'advanced', label: t('stepAdvanced') },
    ];

    const applyPreset = (preset: ChannelPreset) => {
        setState({
            ...state,
            name: state.name || preset.label,
            dialect: preset.dialect,
            base_url: preset.base_url,
            openai_chat_completion_path: preset.openai_chat_completion_path,
            openai_response_path: preset.openai_response_path,
            anthropic_message_path: preset.anthropic_message_path,
        });
        setStep('connection');
    };

    // 新建与编辑都一趟完成且都提交整份配置: 授权按名称引用, 与凭据和模型在同一次请求里原子生效。
    const submit = (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        if (!canSubmit) return;
        const detail = toChannelDetail(state, channel?.id ?? 0);
        if (channel) {
            updateChannel.mutate(detail, { onSuccess: () => setIsOpen(false) });
        } else {
            createChannel.mutate(detail, {
                onSuccess: () => setIsOpen(false),
                onError: (error) => toast.error(error.message),
            });
        }
    };

    // 表单高度固定, 否则切换步骤时弹窗会随内容高度跳动; 内容更高的步骤由步骤区内部滚动消化。
    // 29rem 是连接页恰好铺满所需: 首行留白 8px + 五个字段组 290px + 五道间距 80px + 开关行 20px + 底部按钮 52px。
    // 各步骤首行统一落在同一水平线: connection 与 advanced 的首行是无边框文案, 补 8px 才能与步骤导航的按钮文案对齐;
    // keys 与 grants 的首行是 36px 控件行, 文案居中后天然齐平, 无需补白。步骤区自带 4px 内边距供焦点环显示。
    // calc 一项夹住矮屏, 弹窗不提供滚动, 内容超出视口时底部按钮会点不到。详情视图取同一高度以对齐尺寸。
    return (
        <form onSubmit={submit} className="flex flex-col md:flex-row gap-6 h-[min(29rem,calc(100vh-10rem))]">
            <nav className="md:w-28 shrink-0 flex md:flex-col gap-1 overflow-x-auto pt-1">
                {steps.map((s) => (
                    <button
                        key={s.id}
                        type="button"
                        onClick={() => setStep(s.id)}
                        className={`rounded-xl px-3 py-2 text-left text-sm font-medium whitespace-nowrap transition-colors ${step === s.id ? 'bg-muted' : 'text-muted-foreground hover:bg-muted/50'}`}
                    >
                        {s.label}
                    </button>
                ))}
                {/* 返回统计的入口与步骤同列, 顶到底部与步骤拉开距离, 表明它不是其中一步。
                    窄屏下步骤横向排布, 此时它跟在末尾, mt-auto 不生效。 */}
                {onBack && (
                    <button
                        type="button"
                        onClick={onBack}
                        className="md:mt-auto flex items-center gap-1.5 rounded-xl px-3 py-2 text-left text-sm whitespace-nowrap text-muted-foreground transition-colors hover:bg-muted/50"
                    >
                        <ChevronLeft className="size-4 shrink-0" />
                        {t('backToStats')}
                    </button>
                )}
            </nav>

            <div className="flex-1 min-w-0 flex flex-col min-h-0">
                <div className="flex-1 min-h-0 overflow-y-auto overscroll-contain p-1">
                    {step === 'preset' && (
                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                            {CHANNEL_PRESETS.map((preset) => (
                                <button
                                    key={preset.id}
                                    type="button"
                                    onClick={() => applyPreset(preset)}
                                    className="flex items-center gap-4 rounded-xl border border-border px-5 py-4 hover:bg-muted/50 transition-colors"
                                >
                                    <preset.Icon className={`size-7 shrink-0 ${preset.iconClassName ?? ''}`} />
                                    <span className="min-w-0 text-left">
                                        <span className="block truncate text-lg font-semibold">{preset.label}</span>
                                        <span className="block truncate text-xs text-muted-foreground">{preset.description[locale]}</span>
                                    </span>
                                </button>
                            ))}
                        </div>
                    )}

                    {step === 'connection' && (
                        <div className="space-y-4 pt-2">
                            <div className="space-y-2">
                                <Label htmlFor={`${idPrefix}-name`}>{t('name')}</Label>
                                <Input
                                    id={`${idPrefix}-name`}
                                    value={state.name}
                                    onChange={(e) => setState({ ...state, name: e.target.value })}
                                    className="rounded-xl"
                                    required
                                />
                            </div>
                            <div className="space-y-2">
                                <Label htmlFor={`${idPrefix}-base-url`}>{t('baseUrl')}</Label>
                                <Input
                                    id={`${idPrefix}-base-url`}
                                    type="url"
                                    value={state.base_url}
                                    onChange={(e) => setState({ ...state, base_url: e.target.value })}
                                    className="rounded-xl"
                                    required
                                />
                            </div>
                            {([
                                ['openai_chat_completion_path', 'OpenAI Chat'],
                                ['openai_response_path', 'OpenAI Responses'],
                                ['anthropic_message_path', 'Anthropic'],
                            ] as const).map(([field, label]) => (
                                <div key={field} className="space-y-2">
                                    <Label htmlFor={`${idPrefix}-${field}`}>{label}</Label>
                                    <Input
                                        id={`${idPrefix}-${field}`}
                                        value={state[field]}
                                        onChange={(e) => setState({ ...state, [field]: e.target.value })}
                                        aria-invalid={state[field] !== '' && !state[field].startsWith('/')}
                                        className="rounded-xl font-mono text-sm"
                                    />
                                </div>
                            ))}
                            <div className="flex flex-wrap items-center gap-6">
                                {([['enabled', t('enabled')], ['proxy', t('proxy')]] as const).map(([field, label]) => (
                                    <label key={field} className="flex items-center gap-2 cursor-pointer">
                                        <Switch
                                            checked={state[field]}
                                            onCheckedChange={(checked) => setState({ ...state, [field]: checked })}
                                        />
                                        <span className="text-sm">{label}</span>
                                    </label>
                                ))}
                            </div>
                        </div>
                    )}

                    {step === 'keys' && <FormKeys state={state} setState={setState} />}
                    {step === 'grants' && <FormGrants state={state} setState={setState} />}

                    {step === 'advanced' && (
                        <div className="space-y-4 pt-2">
                            {([
                                ['channel_proxy', t('channelProxy')],
                                ['match_regex', t('matchRegex')],
                            ] as const).map(([field, label]) => (
                                <div key={field} className="space-y-2">
                                    <Label htmlFor={`${idPrefix}-${field}`}>{label}</Label>
                                    <Input
                                        id={`${idPrefix}-${field}`}
                                        value={state[field]}
                                        onChange={(e) => setState({ ...state, [field]: e.target.value })}
                                        className="rounded-xl"
                                    />
                                </div>
                            ))}
                            <FormOverrides state={state} setState={setState} />

                            <FormHeaders state={state} setState={setState} />
                        </div>
                    )}
                </div>

                <div className="shrink-0 flex justify-end gap-2 pt-4">
                    <Button
                        type="button"
                        variant="secondary"
                        onClick={() => setIsOpen(false)}
                        className="rounded-xl h-9 px-4"
                    >
                        {t('cancel')}
                    </Button>
                    <Button type="submit" disabled={isPending || !canSubmit} className="rounded-xl h-9 px-4">
                        {channel ? t('save') : t('submit')}
                    </Button>
                </div>
            </div>
        </form>
    );
}
