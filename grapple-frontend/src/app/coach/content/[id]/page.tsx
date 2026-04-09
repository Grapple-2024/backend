"use client";

import React, { useEffect, useMemo } from "react";
import { ContentPageProvider } from "./context";
import { useMobileContext } from "@/context/mobile";
import DesktopView from "./desktop";
import MobileView from "./mobile";
import VideoDashboardCreateModal from "@/components/VideoDashboardCreateModal";
import { useContentContext } from "@/context/content";
import { useEditSeriesContext } from "@/context/edit-series";

const VideoPageClient: React.FC<any> = () => {
  const { isMobile } = useMobileContext();
  const { isEditing } = useEditSeriesContext();
  const { setOpen } = useContentContext();

  useEffect(() => {
    if (isEditing) {
      setOpen(true);
    }
  }, [isEditing]);
  
  const MemoizedView = useMemo(() => {
    return isMobile ? <MobileView /> : <DesktopView />;
  }, [isMobile]);
  
  return (  
    <ContentPageProvider>
      {MemoizedView}
      <VideoDashboardCreateModal />
    </ContentPageProvider>
  );
};

export default VideoPageClient;