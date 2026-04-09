import React from 'react';
import { Button } from "react-bootstrap";

interface Props {
  onClick?: () => void;
  children: React.ReactNode;
  variant?: ButtonVariants;
  type?: string;
  disabled?: boolean;
}

export enum ButtonVariants {
  PRIMARY = 'primary',
  SECONDARY = 'secondary',
  TERTIARY = 'tertiary',
  SUCCESS = 'success',
  DANGER = 'danger',
  WARNING = 'warning',
  INFO = 'info',
  LIGHT = 'light',
  DARK = 'dark',
  LINK = 'link',
  WHITE = 'white',
  TRANSPARENT = 'transparent',
  DEFAULT = 'primary',
}

const GrappleButton: React.FC<Props> = ({
  onClick,
  children,
  variant = ButtonVariants.PRIMARY,
  type = 'button',
  disabled = false,
}) => {
  const style: any = {
    primary: {
      backgroundColor: 'white', color: '#000', borderColor: '#000', marginRight: '10px', transition: 'all 0.3s',
    },
    secondary: {
      backgroundColor: 'white', color: '#F24B4B', borderColor: '#F24B4B',
    },
    tertiary: {
      backgroundColor: 'black', color: 'white', borderColor: '#F24B4B', width: '200px', marginBottom: '30%',
      ':hover': {
        backgroundColor: '#F24B4B', borderColor: 'black', color: 'black',
      },
    },
    danger: {
      backgroundColor: 'white', color: '#F24B4B', borderColor: '#F24B4B', marginRight: '10px', transition: 'all 0.3s',
      ':hover': {
        backgroundColor: '#F24B4B', color: 'white', borderColor: '#F24B4B',
      },
      ':active': {
        backgroundColor: '#a82b23', color: 'white', borderColor: '#a82b23',
      }
    }
  };

  // Apply dynamic styles based on the 'variant' prop
  const variantStyle = style[variant] || style.primary; // Default to 'primary' style if no variant is specified

  return (
    <Button 
      style={variantStyle}
      onClick={onClick}
      variant="dark" // Bootstrap variant, can be omitted or adjusted as needed
      type={type as any}
      disabled={disabled}
    >
      {children}
    </Button>
  );
}

export default GrappleButton;
