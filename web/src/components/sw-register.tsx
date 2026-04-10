'use client';

import { useEffect } from 'react';
import { SW_MESSAGE_TYPE } from '@/lib/sw';

export function ServiceWorkerRegister() {
    useEffect(() => {
        if (typeof window === 'undefined') return;
        if (!('serviceWorker' in navigator)) return;
        if (process.env.NODE_ENV !== 'production') return;

        let hasRefreshed = false;
        const reloadToRoot = () => {
            if (hasRefreshed) return;
            hasRefreshed = true;
            window.location.assign('/');
        };

        const onControllerChange = () => {
            reloadToRoot();
        };

        const activateUpdate = (registration: ServiceWorkerRegistration) => {
            // First install: no existing controller, so no need to force activation/reload.
            if (!navigator.serviceWorker.controller) return;

            // Prefer `waiting` worker (already installed & waiting to activate).
            const worker = registration.waiting || registration.installing;
            worker?.postMessage({ type: SW_MESSAGE_TYPE.SKIP_WAITING });
        };

        navigator.serviceWorker.addEventListener('controllerchange', onControllerChange);

        navigator.serviceWorker
            .register('/sw.js', { scope: '/' })
            .then((registration) => {
                // If an update is already waiting, activate it immediately.
                if (registration.waiting) {
                    activateUpdate(registration);
                }

                registration.update();
                registration.addEventListener('updatefound', () => {
                    const installing = registration.installing;
                    if (!installing) return;
                    installing.addEventListener('statechange', () => {
                        if (installing.state === 'installed') {
                            // When installed + controller exists => an update is ready (likely in `waiting`)
                            activateUpdate(registration);
                        }
                    });
                });
            })
            .catch(() => {
                // ignore
            });

        return () => {
            navigator.serviceWorker.removeEventListener('controllerchange', onControllerChange);
        };
    }, []);

    return null;
}
