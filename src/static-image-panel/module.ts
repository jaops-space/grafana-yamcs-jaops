import { PanelPlugin } from '@grafana/data';
import { ImagePanelOptions } from './types';
import StaticImagePanel from './components/StaticImagePanel';

export const plugin = new PanelPlugin<ImagePanelOptions>(StaticImagePanel)
    .setPanelOptions((builder) => {
        builder
            .addTextInput({
                name: 'Image URL',
                path: 'imageUrl',
                category: ['Image Source'],
                description: 'URL of the image to display',
            })
            .addSelect({
                name: 'Object Fit',
                path: 'objectFit',
                defaultValue: 'contain',
                category: ['Style'],
                settings: {
                    options: [
                        { value: 'contain', label: 'Contain' },
                        { value: 'cover', label: 'Cover' },
                        { value: 'fill', label: 'Fill' },
                        { value: 'none', label: 'None' },
                        { value: 'scale-down', label: 'Scale Down' },
                    ],
                },
            })
            .addTextInput({
                name: 'Transform',
                path: 'transform',
                category: ['Transforms'],
                description: 'CSS transform string (e.g., translate(10px,20px) rotate(45deg) scale(1.2))',
                defaultValue: '',
            });
    });
