import React, { useState, useEffect, useCallback, useRef } from 'react';
import { PanelProps, GrafanaTheme2 } from '@grafana/data';
import { getBackendSrv, getTemplateSrv } from '@grafana/runtime';
import { Button, Badge, Alert, Spinner, IconButton, useStyles2, Tooltip } from '@grafana/ui';
import { css } from '@emotion/css';
import { PanelOptions, LinkInfo } from '../types';

interface Props extends PanelProps<PanelOptions> {}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    padding: ${theme.spacing(2)};
    height: 100%;
    overflow: auto;
  `,
  linkRow: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: ${theme.spacing(1.5)};
    margin-bottom: ${theme.spacing(1)};
    background: ${theme.colors.background.secondary};
    border-radius: ${theme.shape.radius.default};
  `,
  linkRowActive: css`
    background: ${theme.colors.success.transparent};
    transition: background 0.3s ease-in;
  `,
  linkInfo: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
  `,
  linkName: css`
    font-weight: ${theme.typography.fontWeightMedium};
  `,
  linkDetails: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.secondary};
    margin-left: ${theme.spacing(1)};
  `,
  buttonGroup: css`
    display: flex;
    gap: ${theme.spacing(0.5)};
  `,
  center: css`
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    height: 100%;
    gap: ${theme.spacing(1)};
  `,
});

export const LinksPanel: React.FC<Props> = ({ options, data }) => {
  const styles = useStyles2(getStyles);
  const [links, setLinks] = useState<LinkInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [dsUid, setDsUid] = useState<string | null>(null);
  const [endpoint, setEndpoint] = useState<string | null>(null);
  const [activeLinks, setActiveLinks] = useState<Set<string>>(new Set());
  const prevCounters = useRef<Record<string, { dataIn: string; dataOut: string }>>({});

  // Extract datasource UID and endpoint from the query targets
  useEffect(() => {
    const targets = data.request?.targets as any[];
    
    if (targets && targets.length > 0) {
      const target = targets[0];
      
      // Get datasource UID
      const uid = target.datasource?.uid;
      if (uid) {
        setDsUid(uid);
      }
      
      // Get endpoint - handle variable substitution
      let ep = target.endpoint;
      if (target.asVariable && target.endpointVariable) {
        // Resolve variable
        ep = getTemplateSrv().replace(target.endpointVariable);
      }
      if (ep) {
        setEndpoint(ep);
      }
    }
  }, [data.request?.targets]);

  // Fetch links from the backend
  const fetchLinks = useCallback(async () => {
    if (!dsUid || !endpoint) {
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const url = `/api/datasources/uid/${dsUid}/resources/endpoint/${encodeURIComponent(endpoint)}/links`;
      const result = await getBackendSrv().get(url);
      
      let linksList: LinkInfo[] = Array.isArray(result) ? result : [];
      
      // Apply name filter if configured
      if (options.nameFilter) {
        try {
          const regex = new RegExp(options.nameFilter, 'i');
          linksList = linksList.filter((link) => regex.test(link.name));
        } catch (e) {
          // Invalid regex, ignore filter
        }
      }
      
      // Detect activity by comparing counters with previous poll
      const nowActive = new Set<string>();
      const newCounters: Record<string, { dataIn: string; dataOut: string }> = {};
      for (const link of linksList) {
        const key = link.name;
        const dataIn = String(link.dataInCount ?? '0');
        const dataOut = String(link.dataOutCount ?? '0');
        newCounters[key] = { dataIn, dataOut };
        const prev = prevCounters.current[key];
        if (prev && (prev.dataIn !== dataIn || prev.dataOut !== dataOut)) {
          nowActive.add(key);
        }
      }
      prevCounters.current = newCounters;
      setActiveLinks(nowActive);

      // Clear activity highlight after a short duration
      if (nowActive.size > 0) {
        setTimeout(() => setActiveLinks(new Set()), 1500);
      }

      setLinks(linksList);
    } catch (e: any) {
      setError(e.message || 'Failed to load links');
    } finally {
      setLoading(false);
    }
  }, [dsUid, endpoint, options.nameFilter]);

  // Initial fetch
  useEffect(() => {
    if (dsUid && endpoint) {
      fetchLinks();
    }
    return undefined;
  }, [dsUid, endpoint, fetchLinks]);

  // Auto-refresh
  useEffect(() => {
    if (options.refreshInterval > 0 && dsUid && endpoint) {
      const intervalId = setInterval(fetchLinks, options.refreshInterval * 1000);
      return () => clearInterval(intervalId);
    }
    return undefined;
  }, [fetchLinks, options.refreshInterval, dsUid, endpoint]);

  // Clear success message after 3 seconds
  useEffect(() => {
    if (success) {
      const timer = setTimeout(() => setSuccess(null), 3000);
      return () => clearTimeout(timer);
    }
    return undefined;
  }, [success]);

  // Enable/disable a link
  const toggleLink = async (link: LinkInfo) => {
    if (!dsUid || !endpoint) {
      return;
    }
    
    const action = link.disabled ? 'enable' : 'disable';
    setBusy(link.name);
    
    try {
      await getBackendSrv().post(
        `/api/datasources/uid/${dsUid}/resources/endpoint/${encodeURIComponent(endpoint)}/links/${encodeURIComponent(link.name)}/${action}`
      );
      setSuccess(`${link.name} ${action}d successfully`);
      fetchLinks();
    } catch (e: any) {
      setError(e.message || `Failed to ${action} link`);
    } finally {
      setBusy(null);
    }
  };

  // Reset link counters
  const resetCounters = async (linkName: string) => {
    if (!dsUid || !endpoint) {
      return;
    }
    
    setBusy(linkName);
    
    try {
      await getBackendSrv().post(
        `/api/datasources/uid/${dsUid}/resources/endpoint/${encodeURIComponent(endpoint)}/links/${encodeURIComponent(linkName)}/reset`
      );
      setSuccess(`${linkName} counters reset`);
      fetchLinks();
    } catch (e: any) {
      setError(e.message || 'Failed to reset counters');
    } finally {
      setBusy(null);
    }
  };

  // Show configuration message if not properly set up
  if (!dsUid || !endpoint) {
    return (
      <div className={styles.container}>
        <Alert severity="info" title="Configuration Required">
          Please configure this panel with a Links query:
          <ol style={{ marginTop: '8px', paddingLeft: '20px' }}>
            <li>Select the Yamcs datasource</li>
            <li>Select an endpoint (or use a variable)</li>
            <li>Set Query Type to &quot;Links&quot;</li>
          </ol>
        </Alert>
      </div>
    );
  }

  return (
    <div className={styles.container}>

      {/* Messages */}
      {error && (
        <Alert severity="error" title="Error" onRemove={() => setError(null)}>
          {error}
        </Alert>
      )}
      {success && (
        <Alert severity="success" title="Success">
          {success}
        </Alert>
      )}

      {/* Loading */}
      {loading && links.length === 0 && (
        <div className={styles.center}>
          <Spinner />
          <span>Loading links...</span>
        </div>
      )}

      {/* Empty state */}
      {!loading && links.length === 0 && (
        <Alert severity="info" title="No Links">
          No links found for endpoint &quot;{endpoint}&quot;.
        </Alert>
      )}

      {/* Links list */}
      {links.map((link) => (
        <div key={link.name} className={`${styles.linkRow} ${activeLinks.has(link.name) ? styles.linkRowActive : ''}`}>
          <div className={styles.linkInfo}>
            <span className={styles.linkName}>{link.name}</span>
            <Badge
              text={link.disabled ? 'DISABLED' : link.status || 'OK'}
              color={link.disabled ? 'darkgrey' : link.status === 'OK' ? 'green' : 'red'}
            />
            {options.showDetails && (
              <span className={styles.linkDetails}>
                {link.type} | In: {link.dataInCount ?? 0} | Out: {link.dataOutCount ?? 0}
              </span>
            )}
          </div>
          <div className={styles.buttonGroup}>
            <Button
              size="sm"
              variant={link.disabled ? 'success' : 'destructive'}
              onClick={() => toggleLink(link)}
              disabled={busy === link.name}
            >
              {link.disabled ? 'Enable' : 'Disable'}
            </Button>
            <Tooltip content="Reset counters">
              <IconButton
                name="history-alt"
                size="sm"
                onClick={() => resetCounters(link.name)}
                disabled={busy === link.name}
                aria-label="Reset counters"
              />
            </Tooltip>
          </div>
        </div>
      ))}
    </div>
  );
};
