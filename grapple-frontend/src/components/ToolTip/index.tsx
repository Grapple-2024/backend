import React, { useState } from 'react';
import { FaInfoCircle } from 'react-icons/fa';

function ToolTip({
  text,
  children
}: any) {
  const [showTooltip, setShowTooltip] = useState(false);

  return (
    <div>
      <div 
        onMouseEnter={() => setShowTooltip(true)}
        onMouseLeave={() => setShowTooltip(false)}
        style={{ position: 'relative', display: 'inline-block' }}
      >
        {showTooltip && (
          <div style={{
            position: 'absolute',
            bottom: '100%',
            left: '50%',
            transform: 'translateX(-50%)',
            backgroundColor: '#333',
            color: '#fff',
            padding: '5px',
            borderRadius: '5px',
            zIndex: 1,
          }}>
            {text}
          </div>
        )}
        { children }
      </div>
    </div>
  );
}

export default ToolTip;