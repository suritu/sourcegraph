import classNames from 'classnames'
import React, { useRef, useState } from 'react'
import { Link } from 'react-router-dom'

import { Resizable } from '@sourcegraph/shared/src/components/Resizable'
import { Button } from '@sourcegraph/wildcard'

import { CatalogIcon } from '../../../../../catalog'
import { Badge } from '../../../../../components/Badge'
import { FeedbackPromptContent } from '../../../../../nav/Feedback/FeedbackPrompt'
import { Popover } from '../../../../insights/components/popover/Popover'

import styles from './Sidebar.module.scss'

const SIZE_STORAGE_KEY = 'catalog-sidebar-size'

interface Props {}

export const Sidebar: React.FunctionComponent<Props> = props => (
    <Resizable
        defaultSize={200}
        handlePosition="right"
        storageKey={SIZE_STORAGE_KEY}
        className={styles.resizable}
        element={<SidebarContent className="border-right w-100" {...props} />}
    />
)

const SidebarContent: React.FunctionComponent<Props & { className?: string }> = ({ className, children }) => (
    <div className={classNames('d-flex flex-column', className)}>
        <h2 className="h6 font-weight-bold pt-2 px-2 pb-0 mb-0 d-none">
            <Link to="/catalog" className="d-flex align-items-center text-muted">
                <CatalogIcon className="icon-inline mr-1" /> Catalog
            </Link>
        </h2>
        {children}
        <div className="flex-1" />
        <FeedbackPopoverButton />
    </div>
)

const FeedbackPopoverButton: React.FunctionComponent = () => {
    const buttonReference = useRef<HTMLButtonElement>(null)
    const [isVisible, setVisibility] = useState(false)

    return (
        <div className="d-flex align-items-center px-2">
            <Badge status="wip" className="text-uppercase mr-2" />
            <Button ref={buttonReference} variant="link" size="sm">
                Share feedback
            </Button>
            <Popover
                isOpen={isVisible}
                target={buttonReference}
                onVisibilityChange={setVisibility}
                className={styles.feedbackPrompt}
            >
                <FeedbackPromptContent
                    closePrompt={() => setVisibility(false)}
                    textPrefix="Catalog: "
                    routeMatch="/catalog"
                />
            </Popover>
        </div>
    )
}
