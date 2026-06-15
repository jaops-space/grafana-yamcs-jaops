import React from 'react';

export type LayoutDirection = 'column' | 'row';
export type LayoutJustify = 'flex-start' | 'center' | 'flex-end' | 'space-between' | 'space-around' | 'space-evenly';
export type LayoutAlign = 'stretch' | 'flex-start' | 'center' | 'flex-end';
export type ButtonWidthMode = 'auto' | 'equal' | 'fixed' | 'fill';

export type RuntimeLayoutOptions = {
  layoutDirection?: LayoutDirection;
  layoutWrap?: boolean;
  layoutGap?: number;
  layoutJustify?: LayoutJustify;
  layoutAlign?: LayoutAlign;
  buttonWidthMode?: ButtonWidthMode;
  buttonMinWidth?: number;
  buttonWidth?: number;
  buttonMinHeight?: number;
  buttonHeight?: number;
};

export const DEFAULT_BUTTON_MIN_WIDTH = 160;
export const DEFAULT_BUTTON_MIN_HEIGHT = 40;

export function getRuntimeLayoutStyle(options: RuntimeLayoutOptions): React.CSSProperties {
  return {
    display: 'flex',
    flexDirection: options.layoutDirection ?? 'column',
    flexWrap: (options.layoutWrap ?? true) ? 'wrap' : 'nowrap',
    gap: `${options.layoutGap ?? 4}px`,
    justifyContent: options.layoutJustify ?? 'flex-start',
    alignItems: options.layoutAlign ?? 'stretch',
    padding: '10px',
    width: '100%',
    height: '100%',
    overflow: 'auto',
  };
}

export function getRuntimeButtonWrapperStyle(options: RuntimeLayoutOptions): React.CSSProperties {
  const direction = options.layoutDirection ?? 'column';
  const widthMode = options.buttonWidthMode ?? (direction === 'row' ? 'equal' : 'fill');
  const minWidth = options.buttonMinWidth ?? DEFAULT_BUTTON_MIN_WIDTH;
  const minHeight = options.buttonMinHeight ?? DEFAULT_BUTTON_MIN_HEIGHT;
  const explicitWidth = options.buttonWidth;
  const explicitHeight = options.buttonHeight;

  const base: React.CSSProperties = {
    minWidth,
    minHeight,
    height: explicitHeight ? `${explicitHeight}px` : undefined,
  };

  if (direction === 'column') {
    return {
      ...base,
      width: widthMode === 'auto' ? undefined : explicitWidth ? `${explicitWidth}px` : '100%',
      flex: widthMode === 'auto' ? '0 0 auto' : '0 0 auto',
    };
  }

  if (widthMode === 'fixed') {
    return {
      ...base,
      width: `${explicitWidth ?? minWidth}px`,
      flex: `0 0 ${explicitWidth ?? minWidth}px`,
    };
  }

  if (widthMode === 'fill' || widthMode === 'equal') {
    return {
      ...base,
      flex: `1 1 ${explicitWidth ?? minWidth}px`,
    };
  }

  return {
    ...base,
    flex: '0 0 auto',
  };
}

export function getEditorCardsStyle(): React.CSSProperties {
  return {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(420px, 1fr))',
    alignItems: 'start',
    gap: '8px',
    padding: '8px',
    width: '100%',
  };
}
