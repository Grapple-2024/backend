function isOwnException(ex: any) {
  return ex.message.substring(0, 11) === 'time-number';
}

function leftPadding(value: any, len = 2) {
  const sValue = value.toString();
  const lLen = len - sValue.length;

  let l = '';

  while (l.length < lLen) {
    l += '0';
  }

  return `${l}${sValue}`;
}


function internalTimeFromInt(val: any, { validate, format, leadingZero }: any) {
  const value = parseInt(val, 10);

  if (validate && (value < 0 || value >= 24 * 3600)) {
    throw new RangeError('time-number, timeFromInt(): rangeError, value supposed to be between 0 and 86399');
  }

  let hours = Math.floor(value / 3600);
  const minutes = Math.floor((value - (hours * 3600)) / 60);
  const seconds = value - (hours * 3600) - (minutes * 60);
  let ampm = null;

  if (format === 12 || format === '12') {
    ampm = hours < 12 ? 'AM' : 'PM';

    if (hours === 0) {
      hours = 12;
    } else if (hours > 12) {
      hours -= 12;
    }
  }

  const ret = [leadingZero ? leftPadding(hours) : hours, leftPadding(minutes)];

  if (seconds) {
    ret.push(leftPadding(seconds));
  }

  const timeString = ret.join(':');

  if (ampm) {
    return `${timeString} ${ampm}`;
  }

  return timeString;
}

function compatv1(params: any) {
  if (typeof params === 'boolean') {
    return { validate: params };
  }

  return params;
}

export function timeFromInt(val: any, providedParams = {}) {
  const defaults = { validate: true, format: 24, leadingZero: true };
  const params = Object.assign({}, defaults, compatv1(providedParams));
  const { validate } = params;

  if (!validate) {
    return internalTimeFromInt(val, params);
  }

  try {
    if (((val - parseFloat(val)) + 1) >= 0) {
      return internalTimeFromInt(val, params);
    }

    throw new Error();
  } catch (ex) {
    if (isOwnException(ex)) {
      throw ex;
    }

    throw new Error(`time-number, timeFromInt(): invalud value: '${val}', supposed to be number`);
  }
}

function internalTimeToInt(val: any, { validate }: any) {
  const times = val.split(':');
  const len = times.length;

  while (times.length < 3) {
    times.push('0');
  }

  const iTimes = times.map((i: any) => parseInt(i, 10));

  if (validate) {
    const hours = iTimes[0];

    if (hours < 0 || hours > 23) {
      throw new RangeError(`time-number, timeToInt(): hours must be between 0 and 23, provided value: '${val}'`);
    }

    if (len > 1) {
      const minutes = iTimes[1];

      if (minutes < 0 || minutes > 59) {
        throw new RangeError(`time-number, timeToInt(): minutes must be between 0 and 59, provided value: '${val}'`);
      }
    }

    if (len > 2) {
      const seconds = iTimes[2];

      if (seconds < 0 || seconds > 59) {
        throw new RangeError(`time-number, timeToInt(): seconds must be between 0 and 59, provided value: '${val}'`);
      }
    }
  }

  return (iTimes[0] * 3600) + (iTimes[1] * 60) + iTimes[2];
}

function internal12To24(val: any) {
  if (!val || !val.match) {
    return val;
  }

  if (!val.match(/(am|pm)$/i)) {
    return val;
  }

  if (val.match(/^0+:/)) {
    throw new Error('12h format can\'t have 00:30 AM, it should be 12:30 AM instead');
  }

  if (val.match(/am$/i)) {
    return val.replace(/^(\d+)/, (m: any) => m === '12' ? '0' : m).replace(/\s*am$/i, '');
  }

  return val.replace(/^(\d+)/, (m: any) => m === '12' ? m : (parseInt(m, 10) + 12).toString()).replace(/\s*pm$/i, '');
}

export function timeToInt(value: any, providedParams = {}) {
  const defaults = { validate: true };
  const params = Object.assign({}, defaults, compatv1(providedParams));
  const { validate } = params;

  if (!validate) {
    const val = internal12To24(value);

    return internalTimeToInt(val, params);
  }

  try {
    const val = internal12To24(value);

    if (!val.match(/^\d+(:\d+(:\d+)?)?$/)) {
      throw new Error();
    }

    return internalTimeToInt(val, params);
  } catch (ex) {
    if (isOwnException(ex)) {
      throw ex;
    }

    throw new Error(`time-number, timeToInt(): supported formats are 'HH', 'HH:mm', 'HH:mm:ss', provided value: '${value}' doesn't match any of them`);
  }
}