

export const formatDateToUTCString= (date: Date) => {
  const timezoneOffset = -date.getTimezoneOffset();
  const offsetHours = Math.floor(Math.abs(timezoneOffset) / 60);
  const offsetMinutes = Math.abs(timezoneOffset) % 60;
  const offsetSign = timezoneOffset >= 0 ? '+' : '-';
  
  // Format the offset as a string
  const formattedOffset = `${offsetSign}${String(offsetHours).padStart(2, '0')}:${String(offsetMinutes).padStart(2, '0')}`;
  
  // Format the date as ISO string and remove the 'Z' at the end
  const formattedDate = date.toISOString().replace('Z', formattedOffset);

  return formattedDate;
}