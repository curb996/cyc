import pandas as pd

xls = pd.ExcelFile('config.xlsx')
# 设备
devdf = pd.read_excel(xls, sheet_name='设备')
devdf.to_json('devices.json', orient='records', force_ascii=False, indent=2)
# 点表模板
for sheet in xls.sheet_names:
if sheet != '设备':
df = pd.read_excel(xls, sheet_name=sheet)
df.to_json(f'{sheet}.json', orient='records', force_ascii=False, indent=2)