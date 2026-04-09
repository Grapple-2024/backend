import { Pagination as BSPagination } from 'react-bootstrap';
import { useState } from 'react';

interface PaginationProps {
  count: number;
  onPageChange: (pageNumber: number) => void;
}

const Pagination: React.FC<PaginationProps> = ({ count, onPageChange }) => {
  const [currentPage, setCurrentPage] = useState(1);
  const totalPages = Math.ceil(count / 5); // Assuming 5 items per page

  const handlePageChange = (pageNumber: number) => {
    setCurrentPage(pageNumber);
    onPageChange(pageNumber);
  };

  const paginationItems = [];

  for (let number = 1; number <= totalPages; number++) {
    paginationItems.push(
      <BSPagination.Item key={number} active={number === currentPage} onClick={() => handlePageChange(number)}>
        {number}
      </BSPagination.Item>,
    );
  }

  return (
    <BSPagination>
      <BSPagination.First onClick={() => handlePageChange(1)} />
      <BSPagination.Prev onClick={() => handlePageChange(currentPage > 1 ? currentPage - 1 : 1)} />
      {paginationItems}
      <BSPagination.Next onClick={() => handlePageChange(currentPage < totalPages ? currentPage + 1 : totalPages)} />
      <BSPagination.Last onClick={() => handlePageChange(totalPages)} />
    </BSPagination>
  );
};

export default Pagination;