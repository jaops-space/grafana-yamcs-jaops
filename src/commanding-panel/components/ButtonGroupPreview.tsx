import React from 'react';
import { Card } from '@grafana/ui';
import { getRuntimeButtonWrapperStyle, getRuntimeLayoutStyle } from '../utils/layout';

export function ButtonGroupPreview(props: { options: any; children: React.ReactNode }) {
    const { options, children } = props;

    return (
        <Card style={{ margin: '8px', padding: '12px 14px' }}>
            <Card.Heading>
                <h4 style={{ margin: 0 }}>Group Preview</h4>
            </Card.Heading>
            <Card.Meta>Preview of all runtime buttons with the layout settings below.</Card.Meta>
            <Card.Description>
                <div
                    style={{
                        minHeight: 96,
                        maxHeight: 280,
                        padding: 8,
                        border: '1px solid rgba(204, 204, 220, 0.16)',
                        borderRadius: 4,
                        overflow: 'auto',
                    }}
                >
                    <div style={{ ...getRuntimeLayoutStyle(options), padding: 0, height: 'auto', minHeight: 80 }}>
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
