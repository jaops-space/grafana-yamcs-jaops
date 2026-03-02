import { textUtil } from "@grafana/data";
import { getTemplateSrv } from "@grafana/runtime";
import DOMPurify from "dompurify";
import React from "react";
import { ImagePanelOptions } from "static-image-panel/types";

/** Allowed values for the CSS object-fit property. */
const VALID_OBJECT_FIT = new Set(['fill', 'contain', 'cover', 'none', 'scale-down']);

/**
 * Sanitises a CSS `transform` value by stripping everything that is not a
 * recognised transform function with numeric / unit arguments.
 */
function sanitizeTransform(raw: string): string {
    // Strip any HTML/script content first via DOMPurify
    const clean = DOMPurify.sanitize(raw, { ALLOWED_TAGS: [] });
    // Then allowlist only recognised CSS transform functions
    const allowed = clean.match(
        /\b(translate[XY3d]?|rotate[XYZ3d]?|scale[XY3d]?|skew[XY]?|matrix(3d)?|perspective)\([^)]*\)/gi
    );
    return allowed ? allowed.join(' ') : '';
}

export default function ImageRenderer(options: ImagePanelOptions, url: string) {
    
    const templateSrv = getTemplateSrv();

    const imageUrl = textUtil.sanitizeUrl(templateSrv.replace(url || ''));
    const rawObjectFit = DOMPurify.sanitize(templateSrv.replace(options.objectFit || 'contain'), { ALLOWED_TAGS: [] });
    const objectFit = VALID_OBJECT_FIT.has(rawObjectFit) ? rawObjectFit : 'contain';
    const transform = sanitizeTransform(templateSrv.replace(options.transform || ''));

    if (!imageUrl || imageUrl === 'about:blank') { return <div>No image URL provided</div> };

    const style: React.CSSProperties = {
        width: '100%',
        height: '100%',
        objectFit: objectFit as React.CSSProperties['objectFit'],
        transform,
        transition: 'all 0.3s ease',
    };

    return <img src={imageUrl} alt="Image" style={style} />;

}
