import { useUpdateDisplaySeries } from '@/hook/series';
import React, { useState, useEffect } from 'react';
import { Pagination } from 'react-bootstrap';

interface PaginationProps {
  totalCount: number;
  initialPage?: number;
  isCoach?: boolean;
}

const SeriesPagination: React.FC<PaginationProps> = ({
  totalCount,
  initialPage = 1,
  isCoach = false
}) => {
  const querySeries = useUpdateDisplaySeries();
  const [currentPage, setCurrentPage] = useState(initialPage);
  const pageSize = 6; // Fixed page size
  const totalPages = Math.ceil(totalCount / pageSize);

  useEffect(() => {
    setCurrentPage(initialPage);
  }, [initialPage]);

  const runQuery = (page: number) => {
    if (page < 1 || page > totalPages) return;
    
    setCurrentPage(page);

    const queryParams = { 
      page_size: pageSize,
      page
    };

    querySeries.mutate(queryParams);
  };

  const renderPageNumbers = () => {
    const pageItems = [];
    const maxVisiblePages = 5;

    if (totalPages <= maxVisiblePages) {
      // If total pages are 5 or less, show all pages
      for (let i = 1; i <= totalPages; i++) {
        pageItems.push(
          <Pagination.Item
            key={i}
            active={currentPage === i}
            onClick={() => runQuery(i)}
          >
            {i}
          </Pagination.Item>
        );
      }
    } else {
      // Always show first page
      pageItems.push(
        <Pagination.Item
          key={1}
          active={currentPage === 1}
          onClick={() => runQuery(1)}
        >
          1
        </Pagination.Item>
      );

      if (currentPage > 3) {
        pageItems.push(<Pagination.Ellipsis key="ellipsis1" />);
      }

      // Show current page and one page before and after
      for (let i = Math.max(2, currentPage - 1); i <= Math.min(currentPage + 1, totalPages - 1); i++) {
        pageItems.push(
          <Pagination.Item
            key={i}
            active={currentPage === i}
            onClick={() => runQuery(i)}
          >
            {i}
          </Pagination.Item>
        );
      }

      if (currentPage < totalPages - 2) {
        pageItems.push(<Pagination.Ellipsis key="ellipsis2" />);
      }

      // Always show last page
      pageItems.push(
        <Pagination.Item
          key={totalPages}
          active={currentPage === totalPages}
          onClick={() => runQuery(totalPages)}
        >
          {totalPages}
        </Pagination.Item>
      );
    }

    return pageItems;
  };
  
  return (
    <div style={{
      display: 'flex',
      justifyContent: 'flex-end',
      alignItems: 'flex-end',
      backgroundColor: "#F1F5F9",
      paddingBottom: 20,
    }}>
      <Pagination>
        <Pagination.Prev
          onClick={() => runQuery(currentPage - 1)}
          disabled={currentPage === 1}
        />
        
        {renderPageNumbers()}
        
        <Pagination.Next
          onClick={() => runQuery(currentPage + 1)}
          disabled={currentPage === totalPages}
        />
      </Pagination>
    </div>
  );
};

export default SeriesPagination;