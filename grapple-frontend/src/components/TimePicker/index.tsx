import React                      from 'react';
import { timeFromInt, timeToInt } from '@/util/time-number';
import Select from 'react-select';

function TimePicker({
  end           = '23:59',
  format        = 12,
  initialValue  = '00:00',
  onChange      = (e: any) => {},
  start         = '00:00',
  step          = 30,
  value         = null,
  label         = 'Choose a time',
  ...rest
}) {
  function generateFormattedTime(time: any) {
    const ret = timeFromInt(time, false);

    if (format === 24) {
      return ret;
    }

    const found = ret.match(/^(\d+):/);
    let hour = 1;
    
    if (found) {
      hour  = parseInt(found[1], 10);
    }

    if (hour === 0) {
      return `${ret.replace(/^\d+/, '12')} AM`;
    }

    if (hour < 12) {
      return `${ret} AM`;
    }

    if (hour === 12) {
      return `${ret} PM`;
    }

    const newHour = hour < 22 ? `0${hour - 12}` : (hour - 12).toString();

    return `${ret.replace(/^\d+/, newHour)} PM`;
  }

  function generateTimeRange() {
    const times = [];
    const iend  = timeToInt(end, false);

    for (let i = timeToInt(start, false); i <= iend; i += step * 60) {
      times.push(i);
    }

    return times;
  }

  function listTimeOptions() {
    return generateTimeRange().map((unformattedTime) => {
      const formattedTime = generateFormattedTime(unformattedTime);

      return {
        key: unformattedTime,
        val: formattedTime,
      };
    });
  }

  const timeOptions   = listTimeOptions();
  const optionWidgets = timeOptions.map(({ key, val }) => (
    {
      label: val,
      value: val,
    }
  ));
  
  let currentValue: any = value || initialValue;
  
  try {
    currentValue = timeToInt(currentValue);
  } catch (ex) {
    currentValue = parseInt(currentValue, 10);
  }

  if (!timeOptions.filter(({ key }) => currentValue === key).length) {
    currentValue = timeToInt(start);
  }

  return (
    <>
      <label htmlFor="time-picker" style={{ marginBottom: 10 }}>{label}</label>
      <Select
        name="label"
        options={optionWidgets as any}
        className="basic-multi-select"
        classNamePrefix="select"
        onChange={(e: any) => onChange(e.value)}
      />
    </>
  );
}

export default TimePicker;