import React, { useState, cloneElement } from 'react';
import { Row, Col } from 'react-bootstrap';
import styles from '../sidebar.module.css';

const NodeRow = ({ nodes, active, open, isActive, router, isCoach }: any) => {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null); // Single hover state

  return (
    <>
      {nodes &&
        nodes.map((node: any, index: number) => {
          if (!node.line) {
            const isHovering = hoveredIndex === index; // Check if current node is hovered

            return (
              <Row
                key={index}
                className={`${styles.navLinkSidebar} ${
                  active === node.title
                    ? open
                      ? styles.navLinkSidebarActiveOpen
                      : styles.navLinkSidebarActiveClosed
                    : ''
                } ${!open ? styles.navLinkSidebarIconCollapsed : styles.navLinkSidebarIcon}`}
                id={node?.href}
                onClick={() => {
                  isActive(node.title);
                  if (node?.route) {
                    router.push(`/${isCoach ? 'coach' : 'student'}/${node.route}`);
                  } else if (node?.href) {
                    router.push(`/${isCoach ? 'coach' : 'student'}/settings${node?.href}`);
                  } else {
                    node.cb && node.cb();
                  }
                }}
                onMouseEnter={() => setHoveredIndex(index)} // Set hovered index
                onMouseLeave={() => setHoveredIndex(null)} // Clear hovered index
              >
                <Col
                  xs={open ? 3 : 12}
                  style={
                    !open
                      ? {
                          padding: 0,
                          display: 'flex',
                          justifyContent: 'center',
                          alignItems: 'center',
                          height: '100%',
                        }
                      : { padding: 0 }
                  }
                >
                  {node.Icon && (
                    <div className={styles.nodeIcon}>
                      {cloneElement(node.Icon, {
                        ...node.Icon.props,
                        isHovering, // Pass hover state as a prop
                      })}
                    </div>
                  )}
                </Col>
                {open ? (
                  <Col style={{ padding: '0px 12px' }}>
                    <div className={styles.navLinkTitle}>{node.title}</div>
                  </Col>
                ) : null}
              </Row>
            );
          }
          return null; // Handle fallback
        })}
    </>
  );
};

export default NodeRow;
