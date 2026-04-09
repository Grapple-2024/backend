import { useMutation, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Toast } from "react-bootstrap";

interface WarningProps {
  message: string;
  show: boolean;
  bg: string;
};

const Warning = ({
  message, 
  show,
  bg,
}: WarningProps) => {
  
  return (
    <Toast style={{
      position: 'fixed',
      bottom: '5%',
      right: "5%",
      padding: 10, 
      ...(show ? {} : { display: 'none' })
    }} bg={bg}>
      <Toast.Body style={{ color: 'white' }}>{message}</Toast.Body>
    </Toast>
  );
};

export default Warning;