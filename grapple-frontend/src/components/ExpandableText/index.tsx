import React, { useState } from 'react';
import styles from './ExpandableText.module.css';

interface ExpandableTextProps {
  text: string;
  maxLength: number;
}

const ExpandableText: React.FC<ExpandableTextProps> = ({ text, maxLength }) => {
  const [isExpanded, setIsExpanded] = useState(false);

  const toggleExpand = () => {
    setIsExpanded(!isExpanded);
  };

  const getDisplayText = () => {
    if (text?.length > 0) {
      if (isExpanded || text?.length <= maxLength) {
        return text;
      }
      const truncatedText = text?.substring(0, maxLength);
      const lastSpaceIndex = truncatedText.lastIndexOf(' ');
      return truncatedText.substring(0, lastSpaceIndex) + '...';
    }

    return '';
  };

  return (
    <div className={styles.expandableText}>
      <div className={styles.textContent}>
        {getDisplayText()}
        {text?.length > maxLength && (
          <span onClick={toggleExpand} className={styles.toggleButton}>
            {isExpanded ? 'Show less' : 'Show more'}
          </span>
        )}
      </div>
    </div>
  );
};

export default ExpandableText;
