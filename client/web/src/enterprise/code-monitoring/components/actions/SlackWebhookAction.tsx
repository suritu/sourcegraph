import { gql, useMutation } from '@apollo/client'
import { noop } from 'lodash'
import React, { useCallback, useState } from 'react'

import { Alert, Link, ProductStatusBadge } from '@sourcegraph/wildcard'

import { SendTestSlackWebhookResult, SendTestSlackWebhookVariables } from '../../../../graphql-operations'
import { ActionProps } from '../FormActionArea'

import { ActionEditor } from './ActionEditor'

export const SEND_TEST_SLACK_WEBHOOK = gql`
    mutation SendTestSlackWebhook($namespace: ID!, $description: String!, $slackWebhook: MonitorSlackWebhookInput!) {
        triggerTestSlackWebhookAction(namespace: $namespace, description: $description, slackWebhook: $slackWebhook) {
            alwaysNil
        }
    }
`

export const SlackWebhookAction: React.FunctionComponent<ActionProps> = ({
    action,
    setAction,
    disabled,
    authenticatedUser,
    monitorName,
    _testStartOpen,
}) => {
    const [webhookEnabled, setWebhookEnabled] = useState(action ? action.enabled : true)
    const toggleWebhookEnabled: (enabled: boolean) => void = useCallback(
        enabled => {
            setWebhookEnabled(enabled)
            if (action) {
                setAction({ ...action, enabled })
            }
        },
        [action, setAction]
    )

    const [url, setUrl] = useState(action && action.__typename === 'MonitorSlackWebhook' ? action.url : '')

    const [includeResults, setIncludeResults] = useState(action ? action.includeResults : false)
    const toggleIncludeResults: (includeResults: boolean) => void = useCallback(includeResults => {
        setIncludeResults(includeResults)
    }, [])

    const onSubmit: React.FormEventHandler = useCallback(
        event => {
            event.preventDefault()
            setAction({
                __typename: 'MonitorSlackWebhook',
                id: action ? action.id : '',
                url,
                enabled: webhookEnabled,
                includeResults,
            })
        },
        [action, includeResults, setAction, url, webhookEnabled]
    )

    const onDelete: React.FormEventHandler = useCallback(() => {
        setAction(undefined)
    }, [setAction])

    const [sendTestMessage, { loading, error, called }] = useMutation<
        SendTestSlackWebhookResult,
        SendTestSlackWebhookVariables
    >(SEND_TEST_SLACK_WEBHOOK)

    const onSendTestMessage = useCallback(() => {
        sendTestMessage({
            variables: {
                namespace: authenticatedUser.id,
                description: monitorName,
                slackWebhook: { url, enabled: true, includeResults },
            },
        }).catch(noop) // Ignore errors, they will be handled with the error state from useMutation
    }, [authenticatedUser.id, includeResults, monitorName, sendTestMessage, url])

    const testButtonText = loading
        ? 'Sending message...'
        : called && !error
        ? 'Test message sent!'
        : 'Send test message'

    const testButtonDisabled = !monitorName || !url
    const testButtonDisabledReason = !monitorName
        ? 'Please provide a name for the code monitor before sending a test'
        : !url
        ? 'Please provide a webhook URL before sending a test'
        : undefined

    return (
        <ActionEditor
            title={
                <div className="d-flex align-items-center">
                    Send Slack message to channel <ProductStatusBadge className="ml-1" status="experimental" />{' '}
                </div>
            }
            label="Send Slack message to channel"
            subtitle="Post to a specified Slack channel. Requires webhook configuration."
            idName="slack-webhook"
            disabled={disabled}
            completed={!!action}
            completedSubtitle="Notification will be sent to the specified Slack webhook URL."
            actionEnabled={webhookEnabled}
            toggleActionEnabled={toggleWebhookEnabled}
            includeResults={includeResults}
            toggleIncludeResults={toggleIncludeResults}
            canSubmit={!!url}
            onSubmit={onSubmit}
            onCancel={() => {}}
            canDelete={!!action}
            onDelete={onDelete}
            testButtonDisabled={testButtonDisabled}
            testButtonDisabledReason={testButtonDisabledReason}
            testCalled={called}
            testError={error}
            testLoading={loading}
            testButtonText={testButtonText}
            testAgainButtonText="Send again"
            onTest={onSendTestMessage}
            _testStartOpen={_testStartOpen}
        >
            <Alert variant="info" className="mt-4">
                Go to{' '}
                <Link to="https://api.slack.com/apps" target="_blank" rel="noopener">
                    Slack
                </Link>{' '}
                to create a webhook URL.
                <br />
                <Link to="https://docs.sourcegraph.com/code_monitoring/how-tos/slack" target="_blank" rel="noopener">
                    Read more about how to set up Slack webhooks in the docs.
                </Link>
            </Alert>
            <div className="form-group">
                <label htmlFor="code-monitor-slack-webhook-url">Webhook URL</label>
                <input
                    id="code-monitor-slack-webhook-url"
                    type="url"
                    className="form-control mb-2"
                    data-testid="slack-webhook-url"
                    required={true}
                    onChange={event => {
                        setUrl(event.target.value)
                    }}
                    value={url}
                    autoFocus={true}
                    spellCheck={false}
                />
            </div>
        </ActionEditor>
    )
}
