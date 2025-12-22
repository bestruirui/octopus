import localFont from 'next/font/local';

export const miSans = localFont({
    src: '../../public/MiSans-Normal.woff2',
    variable: '--font-misans',
    display: 'swap',
    preload: true,
});

