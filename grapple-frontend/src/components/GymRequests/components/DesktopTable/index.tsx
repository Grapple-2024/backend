import React, { useMemo, useCallback } from 'react';
import { useQueryClient, useMutation } from "@tanstack/react-query";
import { createColumnHelper, useReactTable, getCoreRowModel, flexRender } from '@tanstack/react-table';
import { Table as BSTable, Button, Spinner } from 'react-bootstrap';
import { useMessagingContext } from '@/context/message';
import { useToken } from '@/hook/user';
import { approveRequest, denyRequest } from '@/api-requests/request';

interface User {
  id: string;
  first_name: string;
  last_name: string;
  requestor_email: string;
  gym_id: string;
  approved?: boolean;
}

interface TableProps {
  defaultData: User[];
  kick?: boolean;
}

const columnHelper = createColumnHelper<User>();

const DesktopTable = ({ defaultData, kick = false }: TableProps) => {
  const data = useMemo(() => defaultData, [defaultData]);

  const token = useToken();
  const queryClient = useQueryClient();
  const { setShow, setColor, setMessage } = useMessagingContext();

  const approveMutation = useMutation({
    mutationKey: ['requests'],
    mutationFn: (id: string) => approveRequest(id, token),
    onSuccess: () => {
      setMessage('Request approved');
      setColor('success');
      setShow(true);

      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['requests'] });
      }, 2000);
    },
    onError: (error: any) => {
      setMessage(error.message);
      setColor('danger');
      setShow(true);
    }
  });

  const denyMutation = useMutation({
    mutationKey: ['requests'],
    mutationFn: (id: string) => denyRequest(id, token),
    onSuccess: () => {
      setMessage('Request Denied');
      setColor('success');
      setShow(true);

      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ['requests'] });
      }, 2000);
    },
    onError: (error: any) => {
      setMessage(error.message);
      setColor('danger');
      setShow(true);
    }
  });

  const handleApprove = useCallback((id: string) => {
    approveMutation.mutate(id);
  }, [approveMutation]);

  const handleDeny = useCallback((id: string) => {
    denyMutation.mutate(id);
  }, [denyMutation]);

  const columns = useMemo(
    () => [
      columnHelper.accessor('first_name', {
        cell: info => info.getValue(),
        header: () => <span>First</span>,
      }),
      columnHelper.accessor('last_name', {
        header: () => <span>Last</span>,
        cell: info => info.getValue(),
      }),
      columnHelper.accessor('requestor_email', {
        header: () => <span>Email</span>,
        cell: info => info.getValue(),
      }),
      columnHelper.accessor('approved', {
        header: () => <></>,
        cell: info => {
          return (
            <>
              {!kick && (
                <Button variant='dark' size='sm' onClick={() => handleApprove(info.row.original.id)}>
                  {"Approve"}
                </Button>
              )}
              <Button
                style={{ marginLeft: 10, backgroundColor: 'white', color: 'black' }}
                variant='dark'
                size='sm'
                onClick={() => handleDeny(info.row.original.id)}
              >
                {(kick ? "Kick" : "Deny")}
              </Button>
            </>
          )
        },
      }),
    ],
    [kick, handleApprove, handleDeny]
  );

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
  });

  return (
    <div className="p-2" style={{ backgroundColor: '#F1F5F9' }}>
      <BSTable striped>
        <thead>
          {table.getHeaderGroups().map(headerGroup => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map(header => (
                <th key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.header, header.getContext())}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
          {table.getRowModel().rows.map(row => (
            <tr key={row.id}>
              {row.getVisibleCells().map(cell => (
                <td key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </BSTable>
    </div>
  );
};

export default DesktopTable;
