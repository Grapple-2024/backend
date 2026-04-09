import { colors } from '@/util/colors';
import React, { useEffect, useState } from 'react';
import { Image } from 'react-bootstrap';

function switchLightDark(inputString: string): string {
  // Check if the string matches the pattern /...-light.svg or /...-dark.svg
  const lightPattern = /^(\/.*)-light\.svg$/;
  const darkPattern = /^(\/.*)-dark\.svg$/;

  if (lightPattern.test(inputString)) {
      return inputString.replace(lightPattern, '$1-dark.svg');
  } else if (darkPattern.test(inputString)) {
      return inputString.replace(darkPattern, '$1-light.svg');
  }

  
  return inputString;
}

const GrappleIcon = ({ 
  variant = 'light',
  width = 35, 
  height = 35,
  src = '/content-light.svg',
  isHovering = false,
}) => {
  const isLight = variant === 'light';
  const [srcContent, setSrcContent] = useState(src);

  useEffect(() => {
    if (isHovering) {
      setSrcContent(switchLightDark(src));
    } else {
      setSrcContent(src);
    }
  }, [isHovering]);
  
  return (
    <div
      style={{ 
        backgroundColor: isLight ? colors.black : '', 
        borderRadius: 6, 
        padding: 5, 
        color: isLight ? 'black' : 'white'
      }}
    >
      <Image src={srcContent} alt="Content" style={{ width, height }}/>
    </div>
  );
};

export default GrappleIcon;
