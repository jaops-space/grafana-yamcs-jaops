import { getTemplateSrv } from "@grafana/runtime";
import React from "react";
import { ImagePanelOptions } from "static-image-panel/types";

export default function ImageRenderer(options: ImagePanelOptions, url: string) {
    
    const templateSrv = getTemplateSrv();

    const imageUrl = templateSrv.replace(url || '');
    const transform = templateSrv.replace(options.transform || '');
    const objectFit = templateSrv.replace(options.objectFit || 'contain') as any;

    if (!imageUrl) { return <div>No image URL provided</div> };

    const style: React.CSSProperties = {
        width: '100%',
        height: '100%',
        objectFit,
        transform,
        transition: 'all 0.3s ease',
    };

    return <img src={imageUrl} alt="Image" style={style} />;

}
