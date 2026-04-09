'use client';


function Layout({ children }: any) {  
  return (
    <div style={{ height: '100vh', maxHeight: '100vh' }}> 
      {children}
    </div>
  )
};

export default Layout;