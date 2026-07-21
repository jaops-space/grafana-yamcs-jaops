import { renderHook, act, waitFor } from '@testing-library/react';
import { AppEvents } from '@grafana/data';

import { useCommandSubmit } from '../useCommandSubmit';
import { getCommandKey } from '../../utils/commandKeys';

const publishMock = jest.fn();
const partialMock = jest.fn();
const replaceMock = jest.fn((v: string) => {
  if (v === '$MODE') {
    return '10';
  }
  return v;
});

jest.mock('@grafana/runtime', () => {
  const actual = jest.requireActual('@grafana/runtime');
  return {
    ...actual,
    getAppEvents: () => ({
      publish: (...args: unknown[]) => publishMock(...args),
    }),
    locationService: {
      partial: (...args: unknown[]) => partialMock(...args),
    },
    getTemplateSrv: () => ({
      replace: (value: string) => replaceMock(value),
    }),
  };
});

describe('useCommandSubmit', () => {
  beforeEach(() => {
    publishMock.mockClear();
    partialMock.mockClear();
    replaceMock.mockClear();
  });

  const commandInfo = {
    endpoint: 'myproject_realtime',
    command: {
      name: 'TestCmd',
      qualifiedName: '/YSS/SIM/TestCmd',
    },
  } as any;

  it('updates variable value in variable mode without calling datasource', () => {
    const commandKey = getCommandKey(commandInfo.command.name, 0);
    const datasource = {
      postResource: jest.fn(),
    } as any;

    const formState = {
      [commandKey]: {
        commandName: '/YSS/SIM/TestCmd',
        arguments: {},
        comment: '',
        variableToSet: 'MODE',
        valueToSet: '2',
        changeMode: 'add',
      },
    } as any;

    const { result } = renderHook(() =>
      useCommandSubmit({
        datasource,
        formState,
        scopedVars: {},
        variableMode: true,
        options: {} as any,
        setLoading: jest.fn(),
        dualCommandInfos: {},
        dualButtonStates: {},
        updateDualButtonStates: jest.fn(),
      })
    );

    act(() => {
      result.current(commandInfo, 0, false);
    });

    expect(partialMock).toHaveBeenCalledWith({ 'var-MODE': 12, replace: true });
    expect(datasource.postResource).not.toHaveBeenCalled();
  });

  it('publishes alert when datasource is missing', () => {
    const commandKey = getCommandKey(commandInfo.command.name, 0);
    const setLoading = jest.fn();

    const formState = {
      [commandKey]: {
        commandName: '/YSS/SIM/TestCmd',
        arguments: {},
        comment: '',
      },
    } as any;

    const { result } = renderHook(() =>
      useCommandSubmit({
        datasource: null,
        formState,
        scopedVars: {},
        variableMode: false,
        options: {} as any,
        setLoading,
        dualCommandInfos: {},
        dualButtonStates: {},
        updateDualButtonStates: jest.fn(),
      })
    );

    act(() => {
      result.current(commandInfo, 0, false);
    });

    expect(setLoading).toHaveBeenCalledWith(false);
    expect(publishMock).toHaveBeenCalledWith({
      type: AppEvents.alertError.name,
      payload: ['Datasource not available'],
    });
  });

  it('issues command and updates dual state on success', async () => {
    const commandKey = getCommandKey(commandInfo.command.name, 0);
    const setLoading = jest.fn();
    const updateDualButtonStates = jest.fn();
    const datasource = {
      postResource: jest.fn().mockResolvedValue(undefined),
    } as any;

    const formState = {
      [commandKey]: {
        commandName: '/YSS/SIM/TestCmd',
        arguments: { a: '1' },
        comment: 'ok',
        isDualButton: true,
        onCommand: {
          commandName: '/YSS/SIM/TestCmdOn',
          arguments: { a: '2' },
          comment: 'on',
        },
      },
    } as any;

    const { result } = renderHook(() =>
      useCommandSubmit({
        datasource,
        formState,
        scopedVars: {},
        variableMode: false,
        options: {} as any,
        setLoading,
        dualCommandInfos: {},
        dualButtonStates: {},
        updateDualButtonStates,
      })
    );

    act(() => {
      result.current(commandInfo, 0, false);
    });

    await waitFor(() => {
      expect(datasource.postResource).toHaveBeenCalledTimes(1);
    });

    expect(datasource.postResource).toHaveBeenCalledWith('endpoint/myproject_realtime/command/issue', {
      name: '/YSS/SIM/TestCmdOn',
      arguments: { a: '2' },
      comment: 'on',
    });
    expect(updateDualButtonStates).toHaveBeenCalledWith({ [commandKey]: 'on' });
    expect(publishMock).toHaveBeenCalledWith({
      type: AppEvents.alertSuccess.name,
      payload: ['Command /YSS/SIM/TestCmdOn issued successfully'],
    });
  });
});
