'use client';

import { AnimatePresence, motion } from 'framer-motion';
import { PageWrapper } from '@/components/common/PageWrapper';
import { useChannelList } from '@/api/endpoints/channel';
import { GRID_CARD_VARIANTS } from '@/lib/animations/fluid-transitions';
import { CreateCard } from './Create';
import { Card } from './Card';

export function Channel() {
    const { data: channelsData } = useChannelList();

    return (
        <PageWrapper>
            {channelsData && (
                <motion.div
                    className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4"
                    variants={GRID_CARD_VARIANTS.container}
                    initial="initial"
                    animate="animate"
                >
                    <motion.div
                        variants={GRID_CARD_VARIANTS.item}
                        initial="initial"
                        animate="animate"
                        layout
                    >
                        <CreateCard />
                    </motion.div>
                    <AnimatePresence mode="popLayout">
                        {[...channelsData].sort((a, b) => a.raw.id - b.raw.id).map((channel) => (
                            <motion.div
                                key={channel.raw.id}
                                variants={GRID_CARD_VARIANTS.item}
                                initial="initial"
                                animate="animate"
                                exit={{
                                    opacity: 0,
                                    filter: 'blur(8px)',
                                    transition: { duration: 0.3 }
                                }}
                                layout
                            >
                                <Card channel={channel.raw} stats={channel.formatted} />
                            </motion.div>
                        ))}
                    </AnimatePresence>
                </motion.div>
            )}
        </PageWrapper>
    );
}
