"""
Sample analytics job — simulates what a real user would write.
Must define a run(df, params) function that takes a pandas DataFrame
and parameters dict, and returns a pandas DataFrame.
"""

import sys

import pandas as pd


def run(df: pd.DataFrame, params: dict) -> pd.DataFrame:
    """
    Analyze sales data by region.
    Expected columns: region, revenue, date
    """
    print(f"analyzing partition with {len(df)} rows, params={params}", file=sys.stderr)

    if "region" in df.columns and "revenue" in df.columns:
        result = (
            df.groupby("region")
            .agg(
                total_revenue=("revenue", "sum"),
                avg_revenue=("revenue", "mean"),
                row_count=("revenue", "count"),
            )
            .reset_index()
        )
        return result

    summary = pd.DataFrame(
        {
            "metric": ["row_count", "col_count"],
            "value": [len(df), len(df.columns)],
        }
    )
    return summary
