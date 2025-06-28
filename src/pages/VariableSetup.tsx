import { PluginPage } from '@grafana/runtime';
import { Card, Alert, Text, Stack } from '@grafana/ui';
import React from 'react';
import VariablesSetupImage from '../img/how-to-use/variables-setup.png';
import RowRepeatImage from '../img/how-to-use/row-repeat.png';
import PanelRepeatImage from '../img/how-to-use/panel-repeat.png';

function VariablePanelSetup() {
    return (
        <PluginPage>
            <p>Welcome! This guide will walk you through setting up and using variables in Grafana to enhance your dashboards with the Yamcs plugin. Variables allow dynamic updates and panel/row repetition for flexible data visualization.</p>

            <Stack direction="row" justifyContent='space-around' alignItems='flex-start'>
                <Stack direction="column" width='60%' gap={2}>

                    <Card>
                        <Card.Heading>Step 1: Create Variables in Dashboard Settings</Card.Heading>
                        <Card.Description>
                            Navigate to your Grafana dashboard, click <Text color='info'>Settings</Text> in the top-right corner, and select <Text color='primary'>Variables</Text>. 
                            Click <Text color='primary'>Add variable</Text>. Set the variable type to <Text color='primary'>Custom</Text>. 
                            In the <Text color='info'>Values</Text> field, enter values separated by commas. For a more user-friendly display, use the format <Text variant='code' color='maxContrast'>Label : actual value</Text>. 
                            For example: <Text variant='code' color='maxContrast'>rover1, rover2, The Best Rover : rover3</Text>. This displays <Text color='warning'>The Best Rover</Text> in the dropdown but uses <Text variant='code' color='maxContrast'>rover3</Text> as the actual value.
                        </Card.Description>
                    </Card>

                    <Card>
                        <Card.Heading>Step 2: Enable Multi-Value or All Options for Repeating</Card.Heading>
                        <Card.Description>
                            To enable panel or row repetition, toggle the <Text color='primary'>Multi-value</Text> or <Text color='primary'>All</Text> option in the variable settings. 
                            This allows panels or rows to repeat for each selected variable value, creating dynamic dashboards.
                        </Card.Description>
                    </Card>

                </Stack>

                <img src={VariablesSetupImage} alt="Variable setup in Grafana" width={600} style={{ display: 'block', margin: '20px auto' }} />

            </Stack>

            <hr/>

            <Card>
                <Card.Heading>Step 3: Use Variables in Panels</Card.Heading>
                <Card.Description>
                    Once the variable is set, a dropdown appears on your dashboard. Changing the dropdown value updates all panels using that variable. 
                    For example, selecting <Text color='warning'>The Best Rover</Text> will use <Text variant='code' color='maxContrast'>rover3</Text> in your queries, dynamically updating the panel data.
                </Card.Description>
            </Card>

            <Stack direction="row" justifyContent="space-between" alignItems='flex-start'>
                <Stack direction="column" width='60%' gap={2}>

                    <Card>
                        <Card.Heading>Step 4: Substitute Variables in Fields</Card.Heading>
                        <Card.Description>
                            All fields in the Yamcs plugin, such as queries or panel settings, support variable substitution using the <Text variant='code' color='maxContrast'>{'${variable}'}</Text> format. 
                            For example, if you have a variable named <Text variant='code' color='maxContrast'>rover</Text>, you can use <Text variant='code' color='maxContrast'>{'${rover}'}</Text> in your query to dynamically insert the selected value, like <Text variant='code' color='maxContrast'>rover3</Text>.
                        </Card.Description>
                    </Card>

                    <Card>
                        <Card.Heading>Step 5: Configure Row Repetition</Card.Heading>
                        <Card.Description>
                            To repeat a row of panels, go to <Text>Add <span style={{ color: '#d8d9da' }}>&gt;</span> Row</Text>. Click the <Text color='info'>settings icon</Text> on the row and select <Text color='primary'>Repeat for</Text>. 
                            Choose the variable you created. If <Text color='primary'>Multi-value</Text> or <Text color='primary'>All</Text> is enabled, the row will repeat for each variable value.
                        </Card.Description>
                    </Card>
                </Stack>

                <img src={RowRepeatImage} alt="Row repetition setup in Grafana" width={500} style={{ display: 'block', margin: '20px auto' }} />

            </Stack>

            <hr/>

            <Stack direction="row" justifyContent="space-between" alignItems='flex-start'>

                <Stack direction="column" width='60%' gap={2}>

                    <Card>
                        <Card.Heading>Step 6: Configure Panel Repetition</Card.Heading>
                        <Card.Description>
                            To repeat individual panels, go to the panel&apos;s settings and navigate to <Text color='primary'>Panel Options <span style={{ color: '#d8d9da' }}>&gt;</span> Repeat Options</Text>. 
                            Select the variable to repeat the panel for each value. Ensure <Text color='primary'>Multi-value</Text> or <Text color='primary'>All</Text> is enabled for repetition to work.
                        </Card.Description>
                    </Card>
                    

                </Stack>

                <img src={PanelRepeatImage} alt="Panel repetition setup in Grafana" width={300} style={{ display: 'block', margin: '20px auto' }} />

            </Stack>

            <Alert title="Tip" severity="info">
                Use variables to create dynamic, reusable dashboards. Combine <Text color='primary'>Multi-value</Text> and <Text color='primary'>All</Text> options with row or panel repetition, and leverage the <Text variant='code' color='maxContrast'>{'${variable}'}</Text> format to make your queries and settings dynamic.
            </Alert>
            

            

            
        </PluginPage>
    );
}

export default VariablePanelSetup;
