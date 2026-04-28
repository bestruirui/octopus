'use client';

import { useState, type ReactNode } from 'react';
import { Bell, Clock, Loader, Pencil, Plus, Power, PowerOff, RefreshCw, Save, Trash2, Webhook, X } from 'lucide-react';
import {
    useAlertRuleList,
    useCreateAlertRule,
    useUpdateAlertRule,
    useDeleteAlertRule,
    useAlertNotifChannelList,
    useCreateNotifChannel,
    useDeleteNotifChannel,
    useUpdateNotifChannel,
    useAlertHistory,
    type AlertRule,
    type AlertNotifChannel,
} from '@/api/endpoints/alert';
import { PageWrapper } from '@/components/common/PageWrapper';
import { Input } from '@/components/ui/input';
import { toast } from 'sonner';
import { useTranslations } from 'next-intl';
import {
    applyAlertChannelDraft,
    applyAlertRuleDraft,
    createAlertChannelDraft,
    createAlertRuleDraft,
    type AlertChannelDraft,
    type AlertRuleDraft,
} from './forms';

const CONDITION_TYPES = ['cost_threshold', 'error_rate', 'quota_exceeded', 'channel_down'] as const;
type ConditionType = (typeof CONDITION_TYPES)[number];

function TabButton({ active, onClick, children }: { active: boolean; onClick: () => void; children: ReactNode }) {
    return (
        <button
            onClick={onClick}
            className={`px-4 py-2 rounded-xl text-sm font-medium transition-all active:scale-95 ${
                active ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground hover:bg-muted/80'
            }`}
        >
            {children}
        </button>
    );
}

export function Alert() {
    const t = useTranslations('alert');
    const [tab, setTab] = useState<'rules' | 'channels' | 'history'>('rules');
    const { data: rules, isLoading: rulesLoading } = useAlertRuleList();
    const { data: channels, isLoading: channelsLoading } = useAlertNotifChannelList();
    const { data: history, isLoading: historyLoading } = useAlertHistory();

    const createRule = useCreateAlertRule();
    const updateRule = useUpdateAlertRule();
    const deleteRule = useDeleteAlertRule();
    const createChannel = useCreateNotifChannel();
    const updateChannel = useUpdateNotifChannel();
    const deleteChannel = useDeleteNotifChannel();

    const [showNewRule, setShowNewRule] = useState(false);
    const [showNewChannel, setShowNewChannel] = useState(false);
    const [editingRuleId, setEditingRuleId] = useState<number | null>(null);
    const [editingChannelId, setEditingChannelId] = useState<number | null>(null);
    const [newRule, setNewRule] = useState<AlertRuleDraft>(() => createAlertRuleDraft());
    const [editingRule, setEditingRule] = useState<AlertRuleDraft>(() => createAlertRuleDraft());
    const [newChannel, setNewChannel] = useState<AlertChannelDraft>(() => createAlertChannelDraft());
    const [editingChannel, setEditingChannel] = useState<AlertChannelDraft>(() => createAlertChannelDraft());

    const getConditionLabel = (conditionType: string) => {
        switch (conditionType) {
            case 'cost_threshold':
                return t('conditions.cost_threshold');
            case 'error_rate':
                return t('conditions.error_rate');
            case 'quota_exceeded':
                return t('conditions.quota_exceeded');
            case 'channel_down':
                return t('conditions.channel_down');
            default:
                return conditionType;
        }
    };

    const getChannelTypeLabel = (channelType: string) => {
        if (channelType === 'webhook') {
            return t('channelTypes.webhook');
        }
        return channelType;
    };

    const getHistoryMessage = (message: string, state: number) => {
        if (message === 'alert triggered') {
            return t('history.messages.triggered');
        }
        if (message === 'alert resolved') {
            return t('history.messages.resolved');
        }
        if (!message) {
            if (state === 1) {
                return t('history.messages.triggered');
            }
            if (state === 2) {
                return t('history.messages.resolved');
            }
        }
        return message;
    };

    const resetNewRule = () => {
        setNewRule(createAlertRuleDraft());
        setShowNewRule(false);
    };

    const resetRuleEdit = () => {
        setEditingRuleId(null);
        setEditingRule(createAlertRuleDraft());
    };

    const resetNewChannel = () => {
        setNewChannel(createAlertChannelDraft());
        setShowNewChannel(false);
    };

    const resetChannelEdit = () => {
        setEditingChannelId(null);
        setEditingChannel(createAlertChannelDraft());
    };

    const handleCreateRule = () => {
        createRule.mutate(newRule, {
            onSuccess: () => {
                toast.success(t('toast.ruleCreated'));
                resetNewRule();
            },
            onError: (e) => toast.error(t('toast.actionFailed'), { description: e.message }),
        });
    };

    const handleToggleRule = (rule: AlertRule) => {
        updateRule.mutate(
            { ...rule, enabled: !rule.enabled },
            {
                onSuccess: () => toast.success(rule.enabled ? t('toast.ruleDisabled') : t('toast.ruleEnabled')),
                onError: (e) => toast.error(t('toast.actionFailed'), { description: e.message }),
            }
        );
    };

    const handleSaveRule = (rule: AlertRule) => {
        updateRule.mutate(applyAlertRuleDraft(rule, editingRule), {
            onSuccess: () => {
                toast.success(t('toast.ruleUpdated'));
                resetRuleEdit();
            },
            onError: (e) => toast.error(t('toast.actionFailed'), { description: e.message }),
        });
    };

    const handleCreateChannel = () => {
        createChannel.mutate(newChannel, {
            onSuccess: () => {
                toast.success(t('toast.channelCreated'));
                resetNewChannel();
            },
            onError: (e) => toast.error(t('toast.actionFailed'), { description: e.message }),
        });
    };

    const handleSaveChannel = (channel: AlertNotifChannel) => {
        updateChannel.mutate(applyAlertChannelDraft(channel, editingChannel), {
            onSuccess: () => {
                toast.success(t('toast.channelUpdated'));
                resetChannelEdit();
            },
            onError: (e) => toast.error(t('toast.actionFailed'), { description: e.message }),
        });
    };

    if (rulesLoading || channelsLoading) {
        return <Loader className="size-6 animate-spin mx-auto mt-12" />;
    }

    return (
        <PageWrapper className="space-y-4 pb-24">
            <div className="flex items-center gap-2 mb-2">
                <TabButton active={tab === 'rules'} onClick={() => setTab('rules')}>{t('tabs.rules')}</TabButton>
                <TabButton active={tab === 'channels'} onClick={() => setTab('channels')}>{t('tabs.channels')}</TabButton>
                <TabButton active={tab === 'history'} onClick={() => setTab('history')}>{t('tabs.history')}</TabButton>
            </div>

            {tab === 'rules' && (
                <div className="rounded-3xl border border-border bg-card p-6 space-y-4">
                    <div className="flex items-center justify-between">
                        <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                            <Bell className="h-5 w-5" />{t('rules.title')}
                        </h2>
                        <button
                            onClick={() => setShowNewRule((prev) => !prev)}
                            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-all active:scale-95"
                        >
                            <Plus className="h-4 w-4" />{t('rules.new')}
                        </button>
                    </div>

                    {showNewRule && (
                        <div className="p-4 rounded-2xl bg-muted/30 border border-border space-y-3">
                            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                                <Input
                                    placeholder={t('rules.form.namePlaceholder')}
                                    value={newRule.name}
                                    onChange={(e) => setNewRule({ ...newRule, name: e.target.value })}
                                    className="rounded-xl"
                                />
                                <select
                                    value={newRule.condition_type}
                                    onChange={(e) => setNewRule({ ...newRule, condition_type: e.target.value as ConditionType })}
                                    className="h-9 px-3 rounded-xl bg-background border border-border text-sm"
                                >
                                    {CONDITION_TYPES.map((ct) => (
                                        <option key={ct} value={ct}>{getConditionLabel(ct)}</option>
                                    ))}
                                </select>
                                <Input
                                    type="number"
                                    placeholder={t('rules.form.thresholdPlaceholder')}
                                    value={newRule.threshold}
                                    onChange={(e) => setNewRule({ ...newRule, threshold: Number(e.target.value) })}
                                    className="rounded-xl"
                                />
                                <select
                                    value={newRule.notif_channel_id}
                                    onChange={(e) => setNewRule({ ...newRule, notif_channel_id: Number(e.target.value) })}
                                    className="h-9 px-3 rounded-xl bg-background border border-border text-sm"
                                >
                                    <option value={0}>{t('rules.form.noChannel')}</option>
                                    {(channels || []).map((ch) => (
                                        <option key={ch.id} value={ch.id}>{ch.name}</option>
                                    ))}
                                </select>
                                <Input
                                    type="number"
                                    min="0"
                                    placeholder={t('rules.form.cooldownPlaceholder')}
                                    value={newRule.cooldown_sec}
                                    onChange={(e) => setNewRule({ ...newRule, cooldown_sec: Number(e.target.value) })}
                                    className="rounded-xl"
                                />
                            </div>
                            <div className="flex gap-2">
                                <button
                                    onClick={handleCreateRule}
                                    className="flex-1 h-9 rounded-xl bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 active:scale-[0.98]"
                                >
                                    {t('actions.create')}
                                </button>
                                <button
                                    onClick={resetNewRule}
                                    className="flex-1 h-9 rounded-xl bg-muted text-muted-foreground text-sm font-medium hover:bg-muted/80 active:scale-[0.98]"
                                >
                                    {t('actions.cancel')}
                                </button>
                            </div>
                        </div>
                    )}

                    <div className="space-y-2">
                        {(rules || []).map((rule) => {
                            const isEditing = editingRuleId === rule.id;

                            return (
                                <div key={rule.id} className="p-3 rounded-xl bg-muted/50 hover:bg-muted transition-colors">
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="flex items-start gap-3 min-w-0 flex-1">
                                            <button
                                                onClick={() => handleToggleRule(rule)}
                                                className={`p-1.5 rounded-lg transition-colors ${
                                                    rule.enabled ? 'text-green-500 bg-green-500/10' : 'text-muted-foreground bg-muted'
                                                }`}
                                            >
                                                {rule.enabled ? <Power className="h-4 w-4" /> : <PowerOff className="h-4 w-4" />}
                                            </button>
                                            {isEditing ? (
                                                <div className="flex-1 space-y-3">
                                                    <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
                                                        <Input
                                                            value={editingRule.name}
                                                            onChange={(e) => setEditingRule({ ...editingRule, name: e.target.value })}
                                                            placeholder={t('rules.form.namePlaceholder')}
                                                            className="rounded-xl"
                                                        />
                                                        <select
                                                            value={editingRule.condition_type}
                                                            onChange={(e) => setEditingRule({ ...editingRule, condition_type: e.target.value as ConditionType })}
                                                            className="h-9 px-3 rounded-xl bg-background border border-border text-sm"
                                                        >
                                                            {CONDITION_TYPES.map((ct) => (
                                                                <option key={ct} value={ct}>{getConditionLabel(ct)}</option>
                                                            ))}
                                                        </select>
                                                        <Input
                                                            type="number"
                                                            value={editingRule.threshold}
                                                            onChange={(e) => setEditingRule({ ...editingRule, threshold: Number(e.target.value) })}
                                                            placeholder={t('rules.form.thresholdPlaceholder')}
                                                            className="rounded-xl"
                                                        />
                                                        <select
                                                            value={editingRule.notif_channel_id}
                                                            onChange={(e) => setEditingRule({ ...editingRule, notif_channel_id: Number(e.target.value) })}
                                                            className="h-9 px-3 rounded-xl bg-background border border-border text-sm"
                                                        >
                                                            <option value={0}>{t('rules.form.noChannel')}</option>
                                                            {(channels || []).map((ch) => (
                                                                <option key={ch.id} value={ch.id}>{ch.name}</option>
                                                            ))}
                                                        </select>
                                                        <Input
                                                            type="number"
                                                            min="0"
                                                            value={editingRule.cooldown_sec}
                                                            onChange={(e) => setEditingRule({ ...editingRule, cooldown_sec: Number(e.target.value) })}
                                                            placeholder={t('rules.form.cooldownPlaceholder')}
                                                            className="rounded-xl"
                                                        />
                                                    </div>
                                                    <div className="flex items-center gap-2">
                                                        <button
                                                            onClick={() => handleSaveRule(rule)}
                                                            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-all active:scale-95"
                                                        >
                                                            <Save className="h-4 w-4" />{t('actions.save')}
                                                        </button>
                                                        <button
                                                            onClick={resetRuleEdit}
                                                            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium bg-muted text-muted-foreground hover:bg-muted/80 transition-all active:scale-95"
                                                        >
                                                            <X className="h-4 w-4" />{t('actions.cancel')}
                                                        </button>
                                                    </div>
                                                </div>
                                            ) : (
                                                <div>
                                                    <div className="font-medium text-sm">{rule.name}</div>
                                                    <div className="text-xs text-muted-foreground">
                                                        {getConditionLabel(rule.condition_type)} &ge; {rule.threshold}
                                                        {rule.cooldown_sec > 0 && ` · ${t('rules.cooldown', { seconds: rule.cooldown_sec })}`}
                                                    </div>
                                                </div>
                                            )}
                                        </div>
                                        <div className="flex items-center gap-2">
                                            {!isEditing ? (
                                                <button
                                                    onClick={() => {
                                                        setEditingRuleId(rule.id);
                                                        setEditingRule(createAlertRuleDraft(rule));
                                                    }}
                                                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium bg-background text-foreground hover:bg-background/80 transition-all active:scale-95"
                                                >
                                                    <Pencil className="h-4 w-4" />{t('actions.edit')}
                                                </button>
                                            ) : null}
                                            <button
                                                onClick={() => {
                                                    if (!confirm(t('rules.confirmDelete'))) return;
                                                    deleteRule.mutate(rule.id, {
                                                        onSuccess: () => {
                                                            if (editingRuleId === rule.id) {
                                                                resetRuleEdit();
                                                            }
                                                        },
                                                        onError: (e) => toast.error(t('toast.actionFailed'), { description: e.message }),
                                                    });
                                                }}
                                                className="p-1.5 rounded-xl text-muted-foreground hover:text-red-500 hover:bg-red-500/10 transition-all active:scale-95"
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                </div>
            )}

            {tab === 'channels' && (
                <div className="rounded-3xl border border-border bg-card p-6 space-y-4">
                    <div className="flex items-center justify-between">
                        <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                            <Webhook className="h-5 w-5" />{t('channels.title')}
                        </h2>
                        <button
                            onClick={() => setShowNewChannel((prev) => !prev)}
                            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-all active:scale-95"
                        >
                            <Plus className="h-4 w-4" />{t('channels.new')}
                        </button>
                    </div>

                    {showNewChannel && (
                        <div className="p-4 rounded-2xl bg-muted/30 border border-border space-y-3">
                            <Input
                                placeholder={t('channels.form.namePlaceholder')}
                                value={newChannel.name}
                                onChange={(e) => setNewChannel({ ...newChannel, name: e.target.value })}
                                className="rounded-xl"
                            />
                            <Input
                                placeholder={t('channels.form.urlPlaceholder')}
                                value={newChannel.url}
                                onChange={(e) => setNewChannel({ ...newChannel, url: e.target.value })}
                                className="rounded-xl"
                            />
                            <Input
                                placeholder={t('channels.form.secretPlaceholder')}
                                value={newChannel.secret}
                                onChange={(e) => setNewChannel({ ...newChannel, secret: e.target.value })}
                                className="rounded-xl"
                            />
                            <div className="flex gap-2">
                                <button
                                    onClick={handleCreateChannel}
                                    className="flex-1 h-9 rounded-xl bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 active:scale-[0.98]"
                                >
                                    {t('actions.create')}
                                </button>
                                <button
                                    onClick={resetNewChannel}
                                    className="flex-1 h-9 rounded-xl bg-muted text-muted-foreground text-sm font-medium hover:bg-muted/80 active:scale-[0.98]"
                                >
                                    {t('actions.cancel')}
                                </button>
                            </div>
                        </div>
                    )}

                    <div className="space-y-2">
                        {(channels || []).map((channel) => {
                            const isEditing = editingChannelId === channel.id;

                            return (
                                <div key={channel.id} className="p-3 rounded-xl bg-muted/50 hover:bg-muted transition-colors">
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="flex items-start gap-3 min-w-0 flex-1">
                                            <Webhook className="h-4 w-4 text-muted-foreground shrink-0 mt-1" />
                                            {isEditing ? (
                                                <div className="flex-1 space-y-3">
                                                    <Input
                                                        value={editingChannel.name}
                                                        onChange={(e) => setEditingChannel({ ...editingChannel, name: e.target.value })}
                                                        placeholder={t('channels.form.namePlaceholder')}
                                                        className="rounded-xl"
                                                    />
                                                    <Input
                                                        value={editingChannel.url}
                                                        onChange={(e) => setEditingChannel({ ...editingChannel, url: e.target.value })}
                                                        placeholder={t('channels.form.urlPlaceholder')}
                                                        className="rounded-xl"
                                                    />
                                                    <Input
                                                        value={editingChannel.secret}
                                                        onChange={(e) => setEditingChannel({ ...editingChannel, secret: e.target.value })}
                                                        placeholder={t('channels.form.secretPlaceholder')}
                                                        className="rounded-xl"
                                                    />
                                                    <div className="flex items-center gap-2">
                                                        <button
                                                            onClick={() => handleSaveChannel(channel)}
                                                            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 transition-all active:scale-95"
                                                        >
                                                            <Save className="h-4 w-4" />{t('actions.save')}
                                                        </button>
                                                        <button
                                                            onClick={resetChannelEdit}
                                                            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium bg-muted text-muted-foreground hover:bg-muted/80 transition-all active:scale-95"
                                                        >
                                                            <X className="h-4 w-4" />{t('actions.cancel')}
                                                        </button>
                                                    </div>
                                                </div>
                                            ) : (
                                                <div>
                                                    <div className="font-medium text-sm">{channel.name}</div>
                                                    <div className="text-xs text-muted-foreground truncate max-w-[300px]">
                                                        {getChannelTypeLabel(channel.type)} · {channel.url}
                                                    </div>
                                                </div>
                                            )}
                                        </div>
                                        <div className="flex items-center gap-2">
                                            {!isEditing ? (
                                                <button
                                                    onClick={() => {
                                                        setEditingChannelId(channel.id);
                                                        setEditingChannel(createAlertChannelDraft(channel));
                                                    }}
                                                    className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-sm font-medium bg-background text-foreground hover:bg-background/80 transition-all active:scale-95"
                                                >
                                                    <Pencil className="h-4 w-4" />{t('actions.edit')}
                                                </button>
                                            ) : null}
                                            <button
                                                onClick={() => {
                                                    if (!confirm(t('channels.confirmDelete'))) return;
                                                    deleteChannel.mutate(channel.id, {
                                                        onSuccess: () => {
                                                            if (editingChannelId === channel.id) {
                                                                resetChannelEdit();
                                                            }
                                                        },
                                                        onError: (e) => toast.error(t('toast.actionFailed'), { description: e.message }),
                                                    });
                                                }}
                                                className="p-1.5 rounded-xl text-muted-foreground hover:text-red-500 hover:bg-red-500/10 transition-all active:scale-95"
                                            >
                                                <Trash2 className="h-4 w-4" />
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                </div>
            )}

            {tab === 'history' && (
                <div className="rounded-3xl border border-border bg-card p-6 space-y-4">
                    <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                        <Clock className="h-5 w-5" />{t('history.title')}
                    </h2>
                    {historyLoading ? (
                        <Loader className="size-6 animate-spin mx-auto mt-4" />
                    ) : (
                        <div className="space-y-2">
                            {(history || []).map((item) => (
                                <div key={item.id} className="flex items-center justify-between gap-3 p-3 rounded-xl bg-muted/50">
                                    <div className="flex items-center gap-3 min-w-0">
                                        <div className={`h-2 w-2 rounded-full shrink-0 ${item.state === 1 ? 'bg-red-500' : item.state === 2 ? 'bg-green-500' : 'bg-muted-foreground'}`} />
                                        <div>
                                            <div className="font-medium text-sm">{item.rule_name}</div>
                                            <div className="text-xs text-muted-foreground">{getHistoryMessage(item.message, item.state)}</div>
                                        </div>
                                    </div>
                                    <span className="text-xs text-muted-foreground shrink-0">
                                        {new Date(item.time).toLocaleString()}
                                    </span>
                                </div>
                            ))}
                            {(!history || history.length === 0) && (
                                <div className="text-center text-sm text-muted-foreground py-8">
                                    <RefreshCw className="h-6 w-6 mx-auto mb-2 opacity-50" />
                                    {t('history.empty')}
                                </div>
                            )}
                        </div>
                    )}
                </div>
            )}
        </PageWrapper>
    );
}
