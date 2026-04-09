'use client';

import Confirm from "@/components/Confirm";
import { useMobileContext } from "@/context/mobile";
import { Suspense } from "react";

const ConfirmPage = () => {  
  const { isMobile } = useMobileContext();

  return (
    <Suspense fallback={<div>Loading...</div>}>
      <div style={{
        marginLeft: isMobile ? 20 : '10%',
        marginRight: isMobile ? 20 : '10%',
        paddingTop: '10%'
      }}> 
        <Confirm profileType="coach"/>
      </div>
    </Suspense>
  );
};

export default ConfirmPage;