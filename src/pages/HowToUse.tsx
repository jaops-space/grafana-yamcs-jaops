import { PluginPage } from '@grafana/runtime';
import { Card, List, Alert, Text } from '@grafana/ui';
import React from 'react';
import StepOneImage from '../img/how-to-use/step-1.png';
import StepTwoImage from '../img/how-to-use/step-2.png';

function HowToUse() {
    return (
        <PluginPage>
            
            <p>Welcome! This guide will help you set up and use the Yamcs Datasource in Grafana. Follow the steps below to get started.</p>

            <Card>
                <Card.Heading>Step 1: Add the Datasource</Card.Heading>
                <Card.Description>
                    Navigate to <Text color='info'>Connections &gt; Data sources</Text>, then click <Text color='info'>Add a new data source</Text>. Search for <Text color='primary'>Yamcs Datasource</Text> and select it.
                </Card.Description>
            </Card>

            <img src={StepOneImage} width={1200} style={{ display: 'block', margin: '20px auto' }}/>

            <Card>
                <Card.Heading>Step 2: Configure the Datasource</Card.Heading>
                <Card.Description>
                    <p>The data source needs to know what to connect to, use the following configuration items for your needs:</p>
                    <List 
                        items={[ 
                            { label: 'Host', description: <>The path to your Yamcs server (e.g., <code>your-yamcs-server:8090</code>).</> },
                            { label: 'Endpoint', description: 'The instance and processor inside that host.' }
                        ]} 
                        renderItem={(item: any) => <><Text color='primary'>{item.label}:</Text> {item.description}</>} 
                    />
                    <p>Once configured, click <Text color='success'>Save & Test</Text> to verify the connection.</p>
                </Card.Description>
            </Card>

            <img src={StepTwoImage} width={1200} style={{ display: 'block', margin: '20px auto' }}/>

            <Alert title="Important" severity='info'>
                For local instances of Yamcs (running at <code>localhost</code>) use <code>host.docker.internal</code> for Windows, and <code>172.17.0.1</code> for Linux, as the host path.
            </Alert>

            <Card>
                <Card.Heading>Step 3: Create a Dashboard</Card.Heading>
                <Card.Description>
                    Go to <Text color='info'>Dashboards &gt; Create Dashboard</Text>, then <Text color='info'>Create Visualization</Text>. Select <Text color='primary'>Yamcs Datasource</Text> as your data source.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 4: Start Querying</Card.Heading>
                <Card.Description>
                    <p>Choose a query type:</p>
                    <List 
                        items={[ 
                            { label: 'Parameter Query', description: 'Fetch real-time or historical telemetry data (Graph, Single value, Discrete value...).' },
                            { label: 'Event Query', description: 'Retrieve system events from Yamcs.' }
                        ]} 
                        renderItem={(item: any) => <><Text color='primary'>{item.label}:</Text> {item.description}</>} 
                    />
                    <p>Search for a parameter and start querying data.</p>
                </Card.Description>
            </Card>

            <Alert title="Tip" severity="info">
                If you&apos;re facing issues, double-check your host and endpoint settings. A correct setup requires at least one host and one endpoint.
            </Alert>

        </PluginPage>
    );
}

export default HowToUse;
