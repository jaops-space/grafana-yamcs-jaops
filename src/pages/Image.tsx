import { PluginPage } from '@grafana/runtime';
import { Card, List, Alert, Text } from '@grafana/ui';
import React from 'react';

function ImagePanelSetup() {
    return (
        <PluginPage>
            
            <p>Welcome! This guide will help you set up and use the Yamcs Image Panel in Grafana. Follow the steps below to get started.</p>

            <Card>
                <Card.Heading>Step 1: Add a New Panel</Card.Heading>
                <Card.Description>
                    Create a new panel by clicking on <Text color='info'>Add Panel</Text>. Set the data source to <Text color='primary'>Yamcs Datasource</Text>.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 2: Choose Image Query Type</Card.Heading>
                <Card.Description>
                    In the query settings, select <Text color='primary'>Image</Text> as the query type.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 3: Configure the Image Panel</Card.Heading>
                <Card.Description>
                    There are two types of image panels you can choose from:
                    <List 
                        items={[ 
                            { label: 'Static Image', description: 'Supports manual URL input for static images.' },
                            { label: 'Telemetric Image', description: 'Queries data as you would for a parameter, but instead shows an image based on the URL provided.' }
                        ]} 
                        renderItem={(item: any) => <><Text color='primary'>{item.label}:</Text> {item.description}</>} 
                    />
                </Card.Description>
            </Card>

            <Alert title="Important" severity="info">
                For Telemetric Image panels, ensure that the URL returned from the query points to a valid image (e.g., a PNG or JPEG).
            </Alert>

        </PluginPage>
    );
}

export default ImagePanelSetup;
