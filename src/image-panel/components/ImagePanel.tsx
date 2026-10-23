import { css } from '@emotion/css';
import { GrafanaTheme2, PanelProps } from '@grafana/data';
import { Alert, Icon, Link, useStyles2 } from '@grafana/ui';
import { ROUTES } from '../../constants';
import React from 'react';
import ImageRenderer from 'static-image-panel/components/ImageRenderer';
import { ImagePanelOptions } from 'static-image-panel/types';
import { prefixRoute } from 'utils/utils.routing';

const getStyles = (theme: GrafanaTheme2) => ({
    emptyState: css`
        padding: ${theme.spacing(2)};
    `,
    waitingState: css`
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: ${theme.spacing(2)};
        color: ${theme.colors.text.secondary};
    `,
});

export default function ImagePanel(props: PanelProps<ImagePanelOptions>) {
    const styles = useStyles2(getStyles);
    const { options, data } = props;
    let images: string[] = [];

    try {
        data.series.forEach((series) => {
            series.fields.forEach((field) => {
                if (field.values.length > 0 && typeof field.values[0] === 'string') {
                    images.push(field.values[field.values.length - 1] as string);
                }
            });
        });
    } catch (ignored) {}

    return (
        <div
            data-testid="jaops-telemetric-image-panel"
            style={{
                width: props.width,
                height: props.height,
                display: 'grid',
                gridTemplateColumns: `repeat(${Math.max(images.length, 1)}, minmax(0, 1fr))`,
                gridTemplateRows: 'minmax(0, 1fr)',
                overflow: 'auto',
            }}
        >
            {images.length === 0 ? (
                options.emptyStateText?.trim() ? (
                    <div className={styles.waitingState} role="status">
                        <Icon name="camera" size="xxl" />
                        <span>{options.emptyStateText}</span>
                    </div>
                ) : (
                    <div className={styles.emptyState}>
                        <Alert severity="info" title="No image received">
                            Select an Image query and wait for an image parameter value.
                            <br />
                            <Link href={prefixRoute(ROUTES.Image)} color="primary">
                                Open Image Panel setup
                            </Link>
                        </Alert>
                    </div>
                )
            ) : (
                images.map((image) => ImageRenderer(options, image))
            )}
        </div>
    );
}
