import { useState } from 'react';
import {
    Trash2,
    CheckCircle2,
    XCircle,
    FileText,
    DollarSign,
    Clock,
    Activity,
    TrendingUp
} from 'lucide-react';
import { useUpdateChannel, useDeleteChannel, type Channel } from '@/api/endpoints/channel';
import {
    MorphingDialogTitle,
    MorphingDialogDescription,
    MorphingDialogClose,
    useMorphingDialog,
} from '@/components/ui/morphing-dialog';
import { Tabs, TabsContents, TabsContent } from '@/components/animate-ui/primitives/animate/tabs';
import { type StatsMetricsFormatted } from '@/api/endpoints/stats';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { ChannelForm, type ChannelFormData } from './Form';

export function CardContent({ channel, stats }: { channel: Channel; stats: StatsMetricsFormatted }) {
    const { setIsOpen } = useMorphingDialog();
    const updateChannel = useUpdateChannel();
    const deleteChannel = useDeleteChannel();
    const [isEditing, setIsEditing] = useState(false);
    const [isConfirmingDelete, setIsConfirmingDelete] = useState(false);
    const [formData, setFormData] = useState<ChannelFormData>({
        name: channel.name,
        type: channel.type,
        enabled: channel.enabled,
        base_url: channel.base_url,
        key: channel.key,
        model: channel.model,
        proxy: channel.proxy,
    });
    const t = useTranslations('channel.detail');

    const currentView = isEditing ? 'editing' : 'viewing';

    const handleUpdate = (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        updateChannel.mutate({ id: channel.id, ...formData }, {
            onSuccess: () => {
                setIsEditing(false);
                setIsOpen(false);
            }
        });
    };

    const handleDeleteClick = () => {
        if (!isConfirmingDelete) {
            setIsConfirmingDelete(true);
            return;
        }

        setIsOpen(false);
        setTimeout(() => {
            deleteChannel.mutate(channel.id);
        }, 300);
    };

    return (
        <>
            <MorphingDialogTitle>
                <header className="mb-6 flex items-center justify-between">
                    <h2 className="text-2xl font-bold text-card-foreground">
                        {isEditing ? t('title.edit') : t('title.view')}
                    </h2>
                    <MorphingDialogClose
                        className="relative top-0 right-0"
                        variants={{
                            initial: { opacity: 0, scale: 0.8 },
                            animate: { opacity: 1, scale: 1 },
                            exit: { opacity: 0, scale: 0.8 }
                        }}
                    />
                </header>
            </MorphingDialogTitle>

            <MorphingDialogDescription>
                <Tabs value={currentView}>
                    <TabsContents>
                        <TabsContent value="viewing" >
                            <div className="max-h-[60vh] overflow-y-auto space-y-4 sm:space-y-5">
                                <dl className="grid gap-3 grid-cols-1 sm:grid-cols-3">
                                    <div className="rounded-2xl border bg-gradient-to-br from-chart-1/10 to-chart-1/5 p-3 sm:p-4">
                                        <dt className="flex items-center gap-2 mb-2 text-xs font-medium text-muted-foreground">
                                            <Activity className="size-4 text-chart-1" />
                                            {t('metrics.totalRequests')}
                                        </dt>
                                        <dd className="text-xl sm:text-2xl font-bold text-chart-1">
                                            {stats.request_count.formatted.value}
                                            <span className="text-xs font-normal ml-1 text-muted-foreground">{stats.request_count.formatted.unit}</span>
                                        </dd>
                                    </div>

                                    <div className="rounded-2xl border bg-gradient-to-br from-chart-3/10 to-chart-3/5 p-3 sm:p-4">
                                        <dt className="flex items-center gap-2 mb-2 text-xs font-medium text-muted-foreground">
                                            <FileText className="size-4 text-chart-3" />
                                            {t('metrics.totalToken')}
                                        </dt>
                                        <dd className="text-xl sm:text-2xl font-bold text-chart-3">
                                            {stats.total_token.formatted.value}
                                            <span className="text-xs font-normal ml-1 text-muted-foreground">{stats.total_token.formatted.unit}</span>
                                        </dd>
                                    </div>

                                    <div className="rounded-2xl border bg-gradient-to-br from-chart-5/10 to-chart-5/5 p-3 sm:p-4">
                                        <dt className="flex items-center gap-2 mb-2 text-xs font-medium text-muted-foreground">
                                            <DollarSign className="size-4 text-chart-5" />
                                            {t('metrics.totalCost')}
                                        </dt>
                                        <dd className="text-xl sm:text-2xl font-bold text-chart-5">
                                            {stats.total_cost.formatted.value}
                                            <span className="text-xs font-normal ml-1 text-muted-foreground">{stats.total_cost.formatted.unit}</span>
                                        </dd>
                                    </div>
                                </dl>

                                {/* 请求详情 */}
                                <section className="space-y-3">
                                    <h4 className="flex items-center gap-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                                        <TrendingUp className="size-3.5" />
                                        {t('sections.requests')}
                                    </h4>
                                    <dl className="grid gap-3 grid-cols-1 sm:grid-cols-2">
                                        <div className="rounded-2xl border bg-card p-3 sm:p-4 transition-colors hover:bg-accent/5">
                                            <dt className="flex items-center gap-2 mb-2 text-xs text-muted-foreground">
                                                <CheckCircle2 className="size-4 text-accent" />
                                                {t('metrics.successRequests')}
                                            </dt>
                                            <dd className="text-2xl font-bold text-accent">
                                                {stats.request_success.formatted.value}
                                                <span className="text-sm font-normal ml-1 text-muted-foreground">{stats.request_success.formatted.unit}</span>
                                            </dd>
                                        </div>

                                        <div className="rounded-2xl border bg-card p-3 sm:p-4 transition-colors hover:bg-accent/5">
                                            <dt className="flex items-center gap-2 mb-2 text-xs text-muted-foreground">
                                                <XCircle className="size-4 text-destructive" />
                                                {t('metrics.failedRequests')}
                                            </dt>
                                            <dd className="text-2xl font-bold text-destructive">
                                                {stats.request_failed.formatted.value}
                                                <span className="text-sm font-normal ml-1 text-muted-foreground">{stats.request_failed.formatted.unit}</span>
                                            </dd>
                                        </div>
                                    </dl>
                                </section>

                                {/* Token 使用 */}
                                <section className="space-y-3">
                                    <h4 className="flex items-center gap-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                                        <FileText className="size-3.5" />
                                        {t('sections.tokens')}
                                    </h4>
                                    <dl className="grid gap-3 grid-cols-1 sm:grid-cols-2">
                                        <div className="rounded-2xl border bg-card p-3 sm:p-4 transition-colors hover:bg-accent/5">
                                            <dt className="flex items-center gap-2 mb-2 text-xs text-muted-foreground">
                                                <div className="size-2 rounded-full bg-chart-1" />
                                                {t('metrics.inputToken')}
                                            </dt>
                                            <dd className="text-2xl font-bold text-card-foreground">
                                                {stats.input_token.formatted.value}
                                                <span className="text-sm font-normal ml-1 text-muted-foreground">{stats.input_token.formatted.unit}</span>
                                            </dd>
                                        </div>

                                        <div className="rounded-2xl border bg-card p-3 sm:p-4 transition-colors hover:bg-accent/5">
                                            <dt className="flex items-center gap-2 mb-2 text-xs text-muted-foreground">
                                                <div className="size-2 rounded-full bg-chart-3" />
                                                {t('metrics.outputToken')}
                                            </dt>
                                            <dd className="text-2xl font-bold text-card-foreground">
                                                {stats.output_token.formatted.value}
                                                <span className="text-sm font-normal ml-1 text-muted-foreground">{stats.output_token.formatted.unit}</span>
                                            </dd>
                                        </div>
                                    </dl>
                                </section>

                                {/* 成本详情 */}
                                <section className="space-y-3">
                                    <h4 className="flex items-center gap-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                                        <DollarSign className="size-3.5" />
                                        {t('sections.costs')}
                                    </h4>
                                    <dl className="grid gap-3 grid-cols-1 sm:grid-cols-2">
                                        <div className="rounded-2xl border bg-card p-3 sm:p-4 transition-colors hover:bg-accent/5">
                                            <dt className="flex items-center gap-2 mb-2 text-xs text-muted-foreground">
                                                <div className="size-2 rounded-full bg-chart-2" />
                                                {t('metrics.inputCost')}
                                            </dt>
                                            <dd className="text-2xl font-bold text-card-foreground">
                                                {stats.input_cost.formatted.value}
                                                <span className="text-sm font-normal ml-1 text-muted-foreground">{stats.input_cost.formatted.unit}</span>
                                            </dd>
                                        </div>

                                        <div className="rounded-2xl border bg-card p-3 sm:p-4 transition-colors hover:bg-accent/5">
                                            <dt className="flex items-center gap-2 mb-2 text-xs text-muted-foreground">
                                                <div className="size-2 rounded-full bg-chart-5" />
                                                {t('metrics.outputCost')}
                                            </dt>
                                            <dd className="text-2xl font-bold text-card-foreground">
                                                {stats.output_cost.formatted.value}
                                                <span className="text-sm font-normal ml-1 text-muted-foreground">{stats.output_cost.formatted.unit}</span>
                                            </dd>
                                        </div>
                                    </dl>
                                </section>

                                {/* 等待时间 */}
                                <dl className="rounded-2xl border bg-card p-3 sm:p-4 transition-colors hover:bg-accent/5">
                                    <dt className="flex items-center gap-2 mb-2 text-xs text-muted-foreground">
                                        <Clock className="size-4 text-primary" />
                                        {t('metrics.avgWaitTime')}
                                    </dt>
                                    <dd className="text-2xl font-bold text-primary">
                                        {stats.wait_time.formatted.value}
                                        <span className="text-sm font-normal ml-1 text-muted-foreground">{stats.wait_time.formatted.unit}</span>
                                    </dd>
                                </dl>
                            </div>

                            {/* 操作按钮 */}
                            <div className="grid gap-3 sm:grid-cols-2 pt-2">
                                <Button
                                    onClick={() => (isConfirmingDelete ? setIsConfirmingDelete(false) : setIsEditing(true))}
                                    variant={isConfirmingDelete ? 'secondary' : 'default'}
                                    className="w-full rounded-2xl h-12"
                                >
                                    {isConfirmingDelete ? t('actions.cancel') : t('actions.edit')}
                                </Button>
                                <Button
                                    onClick={handleDeleteClick}
                                    disabled={deleteChannel.isPending}
                                    variant="destructive"
                                    className="w-full rounded-2xl h-12"
                                >
                                    <Trash2 className={`size-4 transition-transform ${isConfirmingDelete ? 'scale-110' : ''}`} />
                                    {deleteChannel.isPending
                                        ? t('actions.deleting')
                                        : isConfirmingDelete
                                            ? t('actions.confirmDelete')
                                            : t('actions.delete')}
                                </Button>
                            </div>
                        </TabsContent>

                        <TabsContent value="editing">
                            <ChannelForm
                                formData={formData}
                                onFormDataChange={setFormData}
                                onSubmit={handleUpdate}
                                isPending={updateChannel.isPending}
                                submitText={t('actions.save')}
                                pendingText={t('actions.saving')}
                                onCancel={() => setIsEditing(false)}
                                cancelText={t('actions.cancel')}
                                idPrefix="channel"
                            />
                        </TabsContent>
                    </TabsContents>
                </Tabs>
            </MorphingDialogDescription>
        </>
    );
}
