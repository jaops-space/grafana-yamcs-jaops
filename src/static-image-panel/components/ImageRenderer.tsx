import DOMPurify from 'dompurify';
import { getTemplateSrv } from '@grafana/runtime';
import { Alert } from '@grafana/ui';
import { ImagePanelOptions } from 'static-image-panel/types';
import React from 'react';

export default function ImageRenderer(options: ImagePanelOptions, url: string) {
    const templateSrv = getTemplateSrv();

    let imageUrl = DOMPurify.sanitize(templateSrv.replace(url || '')).trim();
    const transform = DOMPurify.sanitize(templateSrv.replace(options.transform || '')).trim();
    const objectFitValue = DOMPurify.sanitize(templateSrv.replace(options.objectFit || 'contain')).trim();
    const allowedObjectFit = new Set(['contain', 'cover', 'fill', 'none', 'scale-down']);
    const objectFit = (allowedObjectFit.has(objectFitValue) ? objectFitValue : 'contain') as React.CSSProperties['objectFit'];

    if (!imageUrl) {
        return (
            <Alert severity="info" title="No image URL">
                Configure an image URL in the panel options.
            </Alert>
        );
    }

    // Support URLs like "example.com/image.png"
    if (/^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/.test(imageUrl) && !imageUrl.startsWith('/')) {
        imageUrl = `https://${imageUrl}`;
    }

    try {
        const parsed = new URL(imageUrl, window.location.origin);

        if (!['http:', 'https:'].includes(parsed.protocol)) {
            return (
                <Alert severity="error" title="Invalid image URL">
                    Use an HTTP or HTTPS image URL.
                </Alert>
            );
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
        return (
            <Alert severity="error" title="Invalid image URL">
                Check the image URL in the panel options.
            </Alert>
        );
    }
}
