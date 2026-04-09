import React from 'react';
import Link from 'next/link';
import styles from './Breadcrumb.module.css';

interface BreadcrumbProps {
  pathArray: string[];
}

const Breadcrumb: React.FC<BreadcrumbProps> = ({ pathArray }) => {
  const generateBreadcrumbs = () => {
    return pathArray.map((path, index) => {
      const routeTo = `/${pathArray.slice(0, index + 1).join('/')}`;
      const isLast = index === pathArray.length - 1;
      
      return (
        <span key={routeTo} className={styles.breadcrumbItem}>
          {!isLast ? (
            <Link href={routeTo}>
              {path.charAt(0).toUpperCase() + path.slice(1)}
            </Link>
          ) : (
            <span>{path.charAt(0).toUpperCase() + path.slice(1)}</span>
          )}
          {!isLast && <span className={styles.separator}> / </span>}
        </span>
      );
    });
  };

  return (
    <nav className={styles.breadcrumb}>
      <Link href="/" style={{ color: 'black'}}>Home</Link>
      {pathArray.length > 0 && <span className={styles.separator}> / </span>}
      {generateBreadcrumbs()}
    </nav>
  );
};

export default Breadcrumb;
