"use client";

import React from "react";
import { ContentPageProvider } from "./context";
import { useMobileContext } from "@/context/mobile";
import DesktopView from "./desktop";
import MobileView from "./mobile";

const VideoPageClient: React.FC<any> = () => {
  const { isMobile } = useMobileContext();

  return (  
    <ContentPageProvider>
      {
        isMobile ? (
          <MobileView />
        ) : (
          <DesktopView />
        )
      }
    </ContentPageProvider>
  );
};

export default VideoPageClient;
