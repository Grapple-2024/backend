import React from 'react';
import styles from './Upgrade.module.css';

const Upgrade = () => {
  return (
    <div className={styles.gradientDiv}>
      <div className={styles.content}>
        <div className={styles.icon}>
          <span role="img" aria-label="crown">👑</span>
        </div>
        <p className={styles.text}>Organize a better gym with</p>
        <h2 className={styles.title}>Grapple Pro</h2>
        <button className={styles.upgradeButton}>Coming Soon...</button>
      </div>
    </div>
  );
};

export default Upgrade;
