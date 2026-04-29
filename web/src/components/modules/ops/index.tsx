'use client';

import { useState } from 'react';
import { Wrench } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { PageWrapper } from '@/components/common/PageWrapper';
import { Tabs, TabsContents, TabsContent, TabsList, TabsTrigger } from '@/components/animate-ui/components/animate/tabs';
import { Cache } from './Cache';
import { Quota } from './Quota';
import { Health } from './Health';
import { System } from './System';
import { Audit } from './Audit';

type OpsTab = 'cache' | 'quota' | 'health' | 'system' | 'audit';

export function Ops() {
    const t = useTranslations('ops');
    const [activeTab, setActiveTab] = useState<OpsTab>('cache');

    return (
        <PageWrapper className="h-full min-h-0 overflow-y-auto overscroll-contain space-y-6 pb-24 md:pb-4 rounded-t-3xl">
            <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
                <div className="flex items-start gap-4">
                    <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                        <Wrench className="h-5 w-5" />
                    </div>
                    <div className="min-w-0 space-y-2">
                        <h2 className="text-2xl font-bold">{t('title')}</h2>
                        <p className="max-w-3xl text-sm leading-6 text-muted-foreground">
                            {t('subtitle')}
                        </p>
                    </div>
                </div>
            </section>

            <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as OpsTab)}>
                <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
                    <div className="overflow-x-auto">
                        <TabsList className="w-max min-w-full xl:min-w-0">
                            <TabsTrigger value="cache">{t('tabs.cache')}</TabsTrigger>
                            <TabsTrigger value="quota">{t('tabs.quota')}</TabsTrigger>
                            <TabsTrigger value="health">{t('tabs.health')}</TabsTrigger>
                            <TabsTrigger value="system">{t('tabs.system')}</TabsTrigger>
                            <TabsTrigger value="audit">{t('tabs.audit')}</TabsTrigger>
                        </TabsList>
                    </div>
                </section>

                <TabsContents>
                    <TabsContent value="cache">
                        <Cache />
                    </TabsContent>
                    <TabsContent value="quota">
                        <Quota />
                    </TabsContent>
                    <TabsContent value="health">
                        <Health />
                    </TabsContent>
                    <TabsContent value="system">
                        <System />
                    </TabsContent>
                    <TabsContent value="audit">
                        <Audit />
                    </TabsContent>
                </TabsContents>
            </Tabs>
        </PageWrapper>
    );
}
