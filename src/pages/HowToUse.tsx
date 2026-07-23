import { PluginPage } from '@grafana/runtime';
import { Card, Alert, Text } from '@grafana/ui';
import React from 'react';
import StepOneImage from '../img/how-to-use/step-1.png';
import StepTwoImage from '../img/how-to-use/step-2.png';

function HowToUse() {
    return (
        <PluginPage>
            <p>
                Welcome! This guide will help you set up and use the Yamcs Datasource in Grafana. Follow the steps below
                to get started.
            </p>

            <Card>
                <Card.Heading>
                    <span data-testid="jaops-setup-page-how-to-use">Step 1: Add the Datasource</span>
                </Card.Heading>
                <Card.Description>
                    Navigate to <Text color="info">Connections &gt; Data sources</Text>, then click{' '}
                    <Text color="info">Add a new data source</Text>. Search for{' '}
                    <Text color="primary">Yamcs Datasource</Text> and select it.
                </Card.Description>
            </Card>

            <img src={StepOneImage} width={1200} style={{ display: 'block', margin: '20px auto' }} />

            <Card>
                <Card.Heading>Step 2: Configure the Datasource</Card.Heading>
                <Card.Description>
                    <p>
                        The data source needs to know what to connect to, use the following configuration items for your
                        needs:
                    </p>
                    <ul>
                        <li>
                            <Text color="primary">Host:</Text> The path to your Yamcs server (e.g.,{' '}
                            <code>your-yamcs-server:8090</code>).
                        </li>
                        <li>
                            <Text color="primary">Endpoint:</Text> The instance and processor inside that host.
                        </li>
                    </ul>
                    <p>
                        Once configured, click <Text color="success">Save &amp; Test</Text> to verify the connection.
                    </p>
                </Card.Description>
            </Card>

            <img src={StepTwoImage} width={1200} style={{ display: 'block', margin: '20px auto' }} />

            <Alert title="Important" severity="info">
                If you are manually running the plugin through the plugin repository Grafana docker container, to refer
                to local instances (running at <code>localhost</code>), either use <code>docker.gateway</code> or the
                proper platform-specific hostname as the <b>host path</b>.
                <br />
                <br />
                The docker container is configured to resolve <code>docker.gateway</code> to{' '}
                <code>host.docker.internal</code> for Windows, and <code>172.17.0.1</code> for Linux.
                <br />
                <br />
                If you are using the plugin in a custom Docker environement, use the proper hostname.
            </Alert>

            <Card>
                <Card.Heading>Step 3: Create a Dashboard</Card.Heading>
                <Card.Description>
                    Go to <Text color="info">Dashboards &gt; Create Dashboard</Text>, then{' '}
                    <Text color="info">Create Visualization</Text>. Select <Text color="primary">Yamcs Datasource</Text>{' '}
                    as your data source.
                </Card.Description>
            </Card>

            <Card>
                <Card.Heading>Step 4: Start Querying</Card.Heading>
                <Card.Description>
                    <p>Choose a query type:</p>
                    <ul>
                        <li>
                            <Text color="primary">Parameter Query:</Text> Fetch real-time or historical telemetry data
                            (Graph, Single value, Discrete value...).
                        </li>
                        <li>
                            <Text color="primary">Event Query:</Text> Retrieve system events from Yamcs.
                        </li>
                    </ul>
                    <p>Search for a parameter and start querying data.</p>
                </Card.Description>
            </Card>

            <Alert title="Tip" severity="info">
                If you&apos;re facing issues, double-check your host and endpoint settings. A correct setup requires at
                least one host and one endpoint.
            </Alert>
        </PluginPage>
    );
}

export default HowToUse;
