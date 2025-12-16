'use client';

import { useState, useMemo, useCallback } from 'react';
import { Plus, Layers } from 'lucide-react';
import { Reorder } from 'motion/react';
import {
    MorphingDialogClose,
    MorphingDialogTitle,
    MorphingDialogDescription,
    useMorphingDialog,
} from '@/components/ui/morphing-dialog';
import { useCreateGroup, type GroupItem, type GroupMode } from '@/api/endpoints/group';
import { useModelChannelList, type LLMChannel } from '@/api/endpoints/model';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Field, FieldLabel, FieldGroup } from '@/components/ui/field';
import { cn } from '@/lib/utils';
import { MemberItem, AddMemberRow, type SelectedMember } from './components';

function MembersSection({
    members,
    onReorder,
    onRemove,
    onWeightChange,
    onAdd,
    removingIds,
    emptyText,
    channels,
    modelChannels,
    showWeight,
}: {
    members: SelectedMember[];
    onReorder: (members: SelectedMember[]) => void;
    onRemove: (id: string) => void;
    onWeightChange: (id: string, weight: number) => void;
    onAdd: (channel: LLMChannel) => void;
    removingIds: Set<string>;
    emptyText: string;
    channels: { id: number; name: string }[];
    modelChannels: LLMChannel[];
    showWeight: boolean;
}) {
    const t = useTranslations('group');
    const [isAdding, setIsAdding] = useState(false);
    const showEmpty = members.filter((m) => !removingIds.has(m.id)).length === 0 && !isAdding;

    return (
        <div className="rounded-xl border border-border/50 bg-muted/30 overflow-hidden">
            {/* 标题栏 */}
            <div className="flex items-center justify-between px-3 py-2 border-b border-border/30 bg-muted/50">
                <span className="text-sm font-medium text-foreground">
                    {t('form.items')}
                    {members.length > 0 && (
                        <span className="ml-1.5 text-xs text-muted-foreground font-normal">
                            ({members.length})
                        </span>
                    )}
                </span>
                <button
                    type="button"
                    onClick={() => setIsAdding(true)}
                    className={cn(
                        'flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-medium transition-colors',
                        isAdding
                            ? 'bg-primary/10 text-primary cursor-default'
                            : 'hover:bg-muted text-muted-foreground hover:text-foreground'
                    )}
                    disabled={isAdding}
                >
                    <Plus className="size-3.5" />
                    <span>{t('form.addItem')}</span>
                </button>
            </div>

            {/* 内容区域 */}
            <div className="relative h-96">
                {/* 空状态 */}
                <div
                    className={cn(
                        'absolute inset-0 flex flex-col items-center justify-center gap-2 text-muted-foreground',
                        'transition-opacity duration-200 ease-out',
                        showEmpty ? 'opacity-100' : 'opacity-0 pointer-events-none'
                    )}
                >
                    <Layers className="size-10 opacity-40" />
                    <span className="text-sm">{emptyText}</span>
                </div>

                {/* 列表区域 */}
                <div
                    className={cn(
                        'h-full overflow-y-auto transition-opacity duration-200',
                        showEmpty ? 'opacity-0' : 'opacity-100'
                    )}
                >
                    <div className="p-2 flex flex-col gap-1.5">
                        {/* 已添加的成员列表 */}
                        {members.length > 0 && (
                            <Reorder.Group
                                axis="y"
                                values={members}
                                onReorder={onReorder}
                                className="flex flex-col gap-1.5"
                            >
                                {members.map((member, index) => (
                                    <MemberItem
                                        key={member.id}
                                        member={member}
                                        onRemove={onRemove}
                                        onWeightChange={onWeightChange}
                                        isRemoving={removingIds.has(member.id)}
                                        index={index}
                                        showWeight={showWeight}
                                    />
                                ))}
                            </Reorder.Group>
                        )}

                        {/* 新增成员项 - 放在列表末尾 */}
                        {isAdding && (
                            <AddMemberRow
                                index={members.length}
                                channels={channels}
                                modelChannels={modelChannels}
                                selectedMembers={members}
                                onConfirm={(ch) => { onAdd(ch); setIsAdding(false); }}
                                onCancel={() => setIsAdding(false)}
                            />
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}

export function CreateDialogContent() {
    const { setIsOpen } = useMorphingDialog();
    const createGroup = useCreateGroup();
    const { data: modelChannels = [] } = useModelChannelList();
    const t = useTranslations('group');

    const [groupName, setGroupName] = useState('');
    const [mode, setMode] = useState<GroupMode>(1);
    const [selectedMembers, setSelectedMembers] = useState<SelectedMember[]>([]);
    const [removingIds, setRemovingIds] = useState<Set<string>>(new Set());

    // 获取唯一的渠道列表
    const channels = useMemo(() => {
        const channelMap = new Map<number, { id: number; name: string }>();
        modelChannels.forEach((mc) => {
            if (!channelMap.has(mc.channel_id)) {
                channelMap.set(mc.channel_id, { id: mc.channel_id, name: mc.channel_name });
            }
        });
        return Array.from(channelMap.values());
    }, [modelChannels]);

    const handleAddMember = (channel: LLMChannel) => {
        setSelectedMembers((prev) => [...prev, { ...channel, id: `${channel.channel_id}-${channel.name}-${Date.now()}`, weight: 1 }]);
    };

    const handleWeightChange = useCallback((id: string, weight: number) => {
        setSelectedMembers((prev) => prev.map((m) => m.id === id ? { ...m, weight } : m));
    }, []);

    const handleRemoveMember = useCallback((id: string) => {
        setRemovingIds((prev) => new Set(prev).add(id));
        setTimeout(() => {
            setSelectedMembers((prev) => prev.filter((m) => m.id !== id));
            setRemovingIds((prev) => { const n = new Set(prev); n.delete(id); return n; });
        }, 200);
    }, []);

    const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();

        const items: GroupItem[] = selectedMembers.map((member, index) => ({
            channel_id: member.channel_id,
            model_name: member.name,
            priority: index + 1,
            weight: member.weight ?? 1,
        }));

        createGroup.mutate(
            {
                name: groupName,
                mode,
                items,
            },
            {
                onSuccess: () => {
                    setGroupName('');
                    setSelectedMembers([]);
                    setIsOpen(false);
                },
            }
        );
    };

    const isValid = groupName.trim() && selectedMembers.length > 0;

    return (
        <>
            <MorphingDialogTitle>
                <header className="mb-5 flex items-center justify-between">
                    <h2 className="text-2xl font-bold text-card-foreground">
                        {t('create.title')}
                    </h2>
                    <MorphingDialogClose
                        className="relative right-0 top-0"
                        variants={{
                            initial: { opacity: 0, scale: 0.8 },
                            animate: { opacity: 1, scale: 1 },
                            exit: { opacity: 0, scale: 0.8 },
                        }}
                    />
                </header>
            </MorphingDialogTitle>
            <MorphingDialogDescription>
                <form onSubmit={handleSubmit}>
                    <FieldGroup className="gap-4">
                        {/* Group Name */}
                        <Field>
                            <FieldLabel htmlFor="group-name">{t('form.name')}</FieldLabel>
                            <Input
                                id="group-name"
                                value={groupName}
                                onChange={(e) => setGroupName(e.target.value)}
                                className="rounded-xl"
                            />
                        </Field>

                        {/* Mode */}
                        <div className="flex gap-1">
                            {([1, 2, 3, 4] as const).map((m) => (
                                <button
                                    key={m}
                                    type="button"
                                    onClick={() => setMode(m)}
                                    className={cn(
                                        'flex-1 py-1 text-xs rounded-lg transition-colors',
                                        mode === m ? 'bg-primary text-primary-foreground' : 'bg-muted hover:bg-muted/80'
                                    )}
                                >
                                    {t(`mode.${m === 1 ? 'roundRobin' : m === 2 ? 'random' : m === 3 ? 'failover' : 'weighted'}`)}
                                </button>
                            ))}
                        </div>

                        {/* 模型区域（包含添加功能） */}
                        <MembersSection
                            members={selectedMembers}
                            onReorder={setSelectedMembers}
                            onRemove={handleRemoveMember}
                            onWeightChange={handleWeightChange}
                            onAdd={handleAddMember}
                            removingIds={removingIds}
                            emptyText={t('form.noItems')}
                            channels={channels}
                            modelChannels={modelChannels}
                            showWeight={mode === 4}
                        />

                        {/* Submit Button */}
                        <Button
                            type="submit"
                            disabled={!isValid || createGroup.isPending}
                            className="w-full rounded-xl h-11"
                        >
                            {createGroup.isPending ? t('create.submitting') : t('create.submit')}
                        </Button>
                    </FieldGroup>
                </form>
            </MorphingDialogDescription>
        </>
    );
}
