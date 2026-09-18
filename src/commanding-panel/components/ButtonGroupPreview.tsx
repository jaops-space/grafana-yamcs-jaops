import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import React from 'react';
import { Card, useStyles2 } from '@grafana/ui';
import { getRuntimeButtonWrapperStyle, getRuntimeLayoutStyle } from '../utils/layout';

const getStyles = (theme: GrafanaTheme2) => ({
    card: css({
        margin: theme.spacing(1),
        padding: `${theme.spacing(1.5)} ${theme.spacing(1.75)}`,
    }),
    title: css({
        margin: 0,
    }),
    preview: css({
        minHeight: theme.spacing(12),
        maxHeight: theme.spacing(35),
        padding: theme.spacing(1),
        border: `1px solid ${theme.colors.border.weak}`,
        borderRadius: theme.shape.radius.default,
        overflow: 'auto',
    }),
    layout: css({
        padding: 0,
        height: 'auto',
        minHeight: theme.spacing(10),
    }),
});

export function ButtonGroupPreview(props: { options: any; children: React.ReactNode }) {
    const { options, children } = props;
    const styles = useStyles2(getStyles);
    const previewLayoutStyle = {
        ...getRuntimeLayoutStyle(options),
        padding: undefined,
        height: undefined,
        minHeight: undefined,
    };

    return (
        <Card className={styles.card}>
            <Card.Heading>
                <h4 className={styles.title}>Group Preview</h4>
            </Card.Heading>
            <Card.Meta>Preview of all runtime buttons with the layout settings below.</Card.Meta>
            <Card.Description>
                <div className={styles.preview}>
                    <div className={styles.layout} style={previewLayoutStyle}>
                        {React.Children.map(children, (child, index) => (
                            <div key={index} style={getRuntimeButtonWrapperStyle(options)}>
                                {child}
                            </div>
                        ))}
                    </div>
                </div>
            </Card.Description>
        </Card>
    );
}
