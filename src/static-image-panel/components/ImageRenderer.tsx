import DOMPurify from 'dompurify';
import { getTemplateSrv } from '@grafana/runtime';
import { ImagePanelOptions } from 'static-image-panel/types';
import React from 'react';

export default function ImageRenderer(options: ImagePanelOptions, url: string) {
    const templateSrv = getTemplateSrv();

    let imageUrl = DOMPurify.sanitize(templateSrv.replace(url || '')).trim();
    const transform = templateSrv.replace(options.transform || '');
    const objectFit = templateSrv.replace(options.objectFit || 'contain') as React.CSSProperties['objectFit'];

    if (!imageUrl) {
        return <div>No image URL provided</div>;
    }

    // Support URLs like "example.com/image.png"
    if (/^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/.test(imageUrl) && !imageUrl.startsWith('/')) {
        imageUrl = `https://${imageUrl}`;
    }

    try {
        const parsed = new URL(imageUrl, window.location.origin);

        if (!['http:', 'https:'].includes(parsed.protocol)) {
            return <div>Invalid image URL</div>;
        }

        const style: React.CSSProperties = {
            width: '100%',
            height: '100%',
            objectFit,
            transform,
            transition: 'all 0.3s ease',
        };

        return <img src={parsed.href} alt="Image" style={style} data-testid="jaops-image-panel-image" />;
    } catch {
        return <div>Invalid image URL</div>;
    }
}
