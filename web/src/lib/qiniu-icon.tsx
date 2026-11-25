import type { SvgIconProps } from '@thesvg/react';

// 七牛云单色图标: 图标库暂未收录, 为简化版(牛角 + 「七」形笔画), 随文字颜色显示。
export default function QiniuIcon(props: SvgIconProps) {
    return (
        <svg viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg" {...props}>
            <path d="M2 2.6c1.6 5 5.2 7.7 10 7.7s8.4-2.7 10-7.7c-2.9 3.5-6.2 5.1-10 5.1S4.9 6.1 2 2.6z" />
            <path d="M8.3 9.4h12.5l-.5 2.3h-8.9l-.8 5.5c-.1.9.3 1.3 1.1 1.3h4.2c.8 0 1.3-.4 1.5-1.2l.3-1.4h2.5l-.4 2.2c-.3 1.7-1.6 2.7-3.4 2.7h-6.3c-1.9 0-2.9-1-2.6-2.9z" />
        </svg>
    );
}
