import { useState } from 'react';
import { Plus } from 'lucide-react';
import {
    MorphingDialog,
    MorphingDialogTrigger,
    MorphingDialogContainer,
    MorphingDialogContent,
    MorphingDialogClose,
    MorphingDialogTitle,
    MorphingDialogDescription,
    useMorphingDialog,
} from '@/components/ui/morphing-dialog';
import { useCreateChannel, ChannelType } from '@/api/endpoints/channel';
import { useTranslations } from 'next-intl';
import { ChannelForm, type ChannelFormData } from './Form';

function CreateCardContent() {
    const { setIsOpen } = useMorphingDialog();
    const createChannel = useCreateChannel();
    const [formData, setFormData] = useState<ChannelFormData>({
        name: '',
        type: ChannelType.OpenAIChat,
        base_url: '',
        key: '',
        model: '',
        enabled: true,
        proxy: false,
    });
    const t = useTranslations('channel.create');

    const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        createChannel.mutate(formData, {
            onSuccess: () => {
                setFormData({
                    name: '',
                    type: ChannelType.OpenAIChat,
                    base_url: '',
                    key: '',
                    model: '',
                    enabled: true,
                    proxy: false,
                });
                setIsOpen(false);
            }
        });
    };

    return (
        <>
            <MorphingDialogTitle>
                <header className="mb-6 flex items-center justify-between">
                    <h2 className="text-2xl font-bold text-card-foreground">{t('dialogTitle')}</h2>
                    <MorphingDialogClose
                        className="relative right-0 top-0"
                        variants={{
                            initial: { opacity: 0, scale: 0.8 },
                            animate: { opacity: 1, scale: 1 },
                            exit: { opacity: 0, scale: 0.8 }
                        }}
                    />
                </header>
            </MorphingDialogTitle>
            <MorphingDialogDescription>
                <ChannelForm
                    formData={formData}
                    onFormDataChange={setFormData}
                    onSubmit={handleSubmit}
                    isPending={createChannel.isPending}
                    submitText={t('submit')}
                    pendingText={t('submitting')}
                    idPrefix="new-channel"
                />
            </MorphingDialogDescription>
        </>
    );
}

export function CreateCard() {
    const t = useTranslations('channel.create');

    return (
        <MorphingDialog>
            <MorphingDialogTrigger className="w-full">
                <article className="h-54 flex flex-col items-center justify-center gap-6 rounded-3xl bg-primary p-6 text-center text-primary-foreground custom-shadow transition-all duration-300 hover:scale-[1.02]">
                    <span className="flex h-16 w-16 items-center justify-center rounded-full bg-primary-foreground/20">
                        <Plus className="h-8 w-8 text-primary-foreground" />
                    </span>
                    <p className="text-lg font-semibold text-primary-foreground">{t('title')}</p>
                </article>
            </MorphingDialogTrigger>

            <MorphingDialogContainer>
                <MorphingDialogContent className="w-full max-w-2xl bg-card text-card-foreground px-4 py-2 rounded-3xl custom-shadow max-h-[90vh] overflow-y-auto">
                    <CreateCardContent />
                </MorphingDialogContent>
            </MorphingDialogContainer>
        </MorphingDialog>
    );
}
