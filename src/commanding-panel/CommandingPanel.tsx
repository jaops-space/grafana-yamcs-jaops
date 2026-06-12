import { useLocationService } from '@grafana/runtime';
import { CommandForms, PanelOptions } from 'commanding-panel/types';
import React, { useState } from 'react';
import { CommandButton } from './components/CommandButton';
import { CommandCard } from './components/CommandCard';
import { VariableRuntime } from './components/VariableRuntime';
import { useCommandInfos } from './hooks/useCommandInfos';
import { useCommandSubmit } from './hooks/useCommandSubmit';
import { useDatasource } from './hooks/useDatasource';
import { useDualButtonStates } from './hooks/useDualButtonStates';
import { useDualCommandInfos } from './hooks/useDualCommandInfos';
import { CommandErrors, CommandingPanelProps } from './types';
import { getCommandKey } from './utils/commandKeys';
import { setArgumentError, validateCommandArgument } from './utils/validation';
import { Card, Field, Combobox, Input } from '@grafana/ui';



export default function CommandingPanel({ variableMode = false, ...props }: CommandingPanelProps) {
	const { data, options, onOptionsChange } = props;
	const location = useLocationService().getLocation();
	const editing = location.search.includes('editPanel=');
	const scopedVars = props.data.request?.scopedVars;
	const datasourceUid = (data.request?.targets?.[0]?.datasource as any)?.uid;

	const datasource = useDatasource(datasourceUid);
	const { dualCommandInfos, fetchDualCommandInfo, clearDualCommandInfo } = useDualCommandInfos(datasource);
	const commandInfos = useCommandInfos({
		datasource,
		targets: data.request?.targets ?? [],
		scopedVars,
		variableMode,
		options,
		fetchDualCommandInfo,
	});

	const [formState, setFormState] = useState<CommandForms>(options.commandForms || {});
	const [errors, setErrors] = useState<CommandErrors>({});
	const [loading, setLoading] = useState(false);
	const { dualButtonStates, updateDualButtonStates } = useDualButtonStates(props.id, options, onOptionsChange);

	const updateLayoutOption = <K extends keyof PanelOptions>(key: K, value: PanelOptions[K]) => {
		onOptionsChange({
			...options,
			[key]: value,
		});
	};

	const layoutStyle: React.CSSProperties = {
		display: 'flex',
		flexDirection: options.layoutDirection ?? (editing ? 'row' : 'column'),
		flexWrap: options.layoutWrap ? 'wrap' : 'nowrap',
		gap: `${options.layoutGap ?? 4}px`,
		justifyContent: options.layoutJustify ?? 'flex-start',
		alignItems: options.layoutAlign ?? 'stretch',
		padding: '10px',
		width: '100%',
		height: editing ? undefined : '100%',
		overflow: editing ? 'auto' : 'hidden',
	};

	const handleArgumentChange = (commandName: string, argName: string, value: any, index: number) => {
		setFormState((prevState) => {
			const commandKey = getCommandKey(commandName, index);
			const newState = {
				...prevState,
				[commandKey]: {
					...prevState[commandKey],
					arguments: {
						...prevState[commandKey]?.arguments,
						[argName]: value,
					},
				},
			};
			onOptionsChange({ ...options, commandForms: newState });
			return newState;
		});
	};

	const handleOptionChange = (commandName: string, option: string, value: any, index: number) => {
		setFormState((prevState) => {
			const commandKey = getCommandKey(commandName, index);
			const newState = {
				...prevState,
				[commandKey]: {
					...prevState[commandKey],
					[option]: value,
				},
			};
			onOptionsChange({ ...options, commandForms: newState });
			return newState;
		});
	};

	const validateArgument = (commandName: string, arg: any, value: any) => {
		setErrors((prev) => setArgumentError(prev, commandName, arg.name, validateCommandArgument(arg, value)));
	};

	const handleSubmit = useCommandSubmit({
		datasource,
		formState,
		scopedVars,
		variableMode,
		options,
		setLoading,
		dualCommandInfos,
		dualButtonStates,
		updateDualButtonStates,
	});

	let Wrapper = ({children}: {children: React.ReactNode}) => {
		return <div
			style={{
				flex: options.equalButtonWidth ? '1 1 240px' : '0 0 auto',
				minWidth: options.equalButtonWidth ? 240 : undefined,
				width: options.layoutDirection === 'column' ? '100%' : undefined,
			}}
			>
			{children}
		</div>
	}

	return (
		<div style={{ width: '100%', height: '100%', overflow: editing ? 'scroll' : 'unset' }}>
			<div style={layoutStyle}>
				{commandInfos.map((commandInfo, index) => {
					const command = commandInfo.command;
					const commandState = formState[getCommandKey(command.name, index)];

					if (!editing) {
						if (variableMode) {
							return (
								<Wrapper key={command.name + index}>
									<VariableRuntime
										key={getCommandKey(command.name, index)}
										commandInfo={commandInfo}
										index={index}
										commandState={commandState}
										scopedVars={scopedVars}
										loading={loading}
										dualButtonStates={dualButtonStates}
										onSubmit={handleSubmit}
									/>
								</Wrapper>
							);
						}

						return (
							<Wrapper key={command.name + index}>
								<CommandButton
									key={getCommandKey(command.name, index)}
									commandInfo={commandInfo}
									index={index}
									commandState={commandState}
									scopedVars={scopedVars}
									loading={loading}
									dualButtonStates={dualButtonStates}
									onSubmit={handleSubmit}
								/>
							</Wrapper>
						);
					}

					return (
						<CommandCard
							key={getCommandKey(command.name, index)}
							commandInfo={commandInfo}
							index={index}
							commandState={commandState}
							variableMode={variableMode}
							scopedVars={scopedVars}
							loading={loading}
							datasource={datasource}
							errors={errors}
							dualCommandInfos={dualCommandInfos}
							dualButtonStates={dualButtonStates}
							onSubmit={handleSubmit}
							onArgumentChange={handleArgumentChange}
							onOptionChange={handleOptionChange}
							onValidate={validateArgument}
							fetchDualCommandInfo={fetchDualCommandInfo}
							clearDualCommandInfo={clearDualCommandInfo}
						/>
					);
				})}
			</div>
			{editing && commandInfos.length > 1 && (
			<Card style={{ margin: '10px', padding: '12px' }}>
				<Card.Heading>
				<h4 style={{ margin: 0 }}>Button Layout</h4>
				</Card.Heading>

				<Card.Description>
				<div
					style={{
					display: 'grid',
					gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
					gap: '10px',
					alignItems: 'end',
					}}
				>
					<Field label="Direction">
					<Combobox
						value={options.layoutDirection ?? (editing ? 'row' : 'column')}
						options={[
						{ label: 'Vertical', value: 'column' },
						{ label: 'Horizontal', value: 'row' },
						]}
						onChange={(e) => updateLayoutOption('layoutDirection', e.value as any)}
					/>
					</Field>

					<Field label="Wrap">
					<Combobox
						value={String(options.layoutWrap ?? true)}
						options={[
						{ label: 'No wrap', value: 'false' },
						{ label: 'Wrap', value: 'true' },
						]}
						onChange={(e) => updateLayoutOption('layoutWrap', e.value === 'true')}
					/>
					</Field>

					<Field label="Gap">
					<Input
						type="number"
						min={0}
						value={options.layoutGap ?? 4}
						onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
						updateLayoutOption('layoutGap', Number(e.currentTarget.value))
						}
					/>
					</Field>

					<Field label="Justify">
					<Combobox
						value={options.layoutJustify ?? 'flex-start'}
						options={[
						{ label: 'Start', value: 'flex-start' },
						{ label: 'Center', value: 'center' },
						{ label: 'End', value: 'flex-end' },
						{ label: 'Space between', value: 'space-between' },
						{ label: 'Space around', value: 'space-around' },
						]}
						onChange={(e) => updateLayoutOption('layoutJustify', e.value as any)}
					/>
					</Field>

					<Field label="Align">
					<Combobox
						value={options.layoutAlign ?? 'stretch'}
						options={[
						{ label: 'Stretch', value: 'stretch' },
						{ label: 'Start', value: 'flex-start' },
						{ label: 'Center', value: 'center' },
						{ label: 'End', value: 'flex-end' },
						]}
						onChange={(e) => updateLayoutOption('layoutAlign', e.value as any)}
					/>
					</Field>

					<Field label="Equal width">
					<Combobox
						value={String(options.equalButtonWidth ?? true)}
						options={[
						{ label: 'Auto', value: 'false' },
						{ label: 'Equal', value: 'true' },
						]}
						onChange={(e) => updateLayoutOption('equalButtonWidth', e.value === 'true')}
					/>
					</Field>
				</div>
				</Card.Description>
			</Card>
			)}
		</div>
	);
}
